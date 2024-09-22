package exhentai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

func (c exhentaiImporter) doGetJSONRequest(ctx context.Context, path string, dest any) error {
	return c.doGetRequest(ctx, path,
		func(h http.Header) {
			h.Set("accept", "application/json, text/plain, */*")
		},
		func(body io.ReadCloser) error {
			return json.NewDecoder(body).Decode(dest)
		},
	)
}

func (c exhentaiImporter) doGetRequest(ctx context.Context, path string, setHeader func(http.Header), handler func(io.ReadCloser) error) error {
	const maxAttempts = 5
	for i := 0; i < maxAttempts; i++ {
		reqCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, path, nil)
		if err != nil {
			return err
		}

		setHeader(req.Header)

		res, err := c.client.Do(req)
		if err != nil {
			return err
		}

		if res.StatusCode == 429 && (i < maxAttempts-1) {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(8 * time.Second * time.Duration(math.Pow(1.8, float64(i)))):
			}
			continue
		}

		defer func() {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}()

		if res.StatusCode < 200 || res.StatusCode > 299 {
			buf := &bytes.Buffer{}
			io.Copy(buf, res.Body)
			return util.NewStatusCodeError(res.StatusCode, buf.String())
		}

		return handler(res.Body)
	}

	return errors.New("too many failed attempts")
}

func (c exhentaiImporter) createFileDownloadOp(ctx context.Context, destFolder string, urlString string, origFilename string, onCompleteFn func(context.Context, string, string, []byte) error) (*downloader.DownloadOp, error) {
	fn, err := util.StoreFilename(origFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to determine file name to save: %w", err)
	}
	dest := path.Join(destFolder, fn)

	log.Debugf("Saving file to '%s'", dest)

	req, err := http.NewRequestWithContext(ctx, "GET", exhentaiDataBaseUrl+urlString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	op := downloader.NewDownloadOp(dest, req)
	op.OnComplete = func(hash []byte) error {
		found, err := c.findDuplicate(ctx, hash, urlString, dest)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Errorf("Failed to search for duplicate file: %v", err)
			}
		} else if found != "" {
			dest = found
		}

		return onCompleteFn(ctx, dest, urlString+"?orig="+origFilename, hash)
	}

	return op, nil
}

func (c exhentaiImporter) findDuplicate(ctx context.Context, hash []byte, remotePath string, dest string) (string, error) {
	if len(hash) == 0 {
		return "", nil
	}

	mid, loc, err := models.FindMediaByHash(ctx, hash)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	if loc == "" {
		return "", nil
	}

	log.Infof("Downloaded file '%s' was already found at path '%s' with matching hash (media %v)", remotePath, loc, mid)

	err = fs.SharedFS.Delete(ctx, dest)
	if err != nil {
		return "", fmt.Errorf("failed to delete duplicate file: %w", err)
	}

	return loc, nil
}

func mediaID(post *models.Post, idx int) string {
	return post.SourceID + "-" + strconv.Itoa(idx)
}
