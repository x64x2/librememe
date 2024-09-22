package exhentai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

)

const baseURL = "https://gelbooru.com"

type Scraper struct {
	ForceRescan        bool
	ProfilesBySourceId map[string]ScrapeProfile

	client *http.Client
}

func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 1 * time.Minute,
				}).Dial,
				TLSHandshakeTimeout:   30 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

func (s *Scraper) createFileDownloadOp(ctx context.Context, destFolder string, mediaID int64, urlString string, onCompleteFn func(context.Context, string, string, []byte) error) (*downloader.DownloadOp, error) {
	requrl, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("invalid request URL '%s': %w", urlString, err)
	}

	remotePath := util.RemoveQS(urlString)

	legacyFn := util.LegacyStoreFilename(requrl.Path, mediaID)
	legacyDest := path.Join(destFolder, legacyFn)
	existSize, err := fs.SharedFS.Stat(ctx, legacyDest)
	if err != nil {
		return nil, fmt.Errorf("failed to stat for file '%s': %w", legacyDest, err)
	}
	if existSize > 0 {
		log.Debugf("File '%s' already exists at legacy path", legacyDest)
		err = onCompleteFn(ctx, legacyDest, remotePath, nil)
		return nil, err
	}

	fn, err := util.StoreFilename(requrl.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to determine file name to save: %w", err)
	}
	dest := path.Join(destFolder, fn)

	log.Debugf("Saving file to '%s'", dest)

	req, err := http.NewRequestWithContext(ctx, "GET", urlString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("user-agent", config.Auth.GetString("exhentai.UserAgent"))
	req.Header.Set("Referer", "https://gelbooru.com/")

	op := downloader.NewDownloadOp(dest, req)
	op.OnComplete = func(hash []byte) error {
		found, err := s.findDuplicate(ctx, hash, remotePath, dest)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Errorf("Failed to search for duplicate file: %v", err)
			}
		} else if found != "" {
			dest = found
		}

		return onCompleteFn(ctx, dest, remotePath, hash)
	}

	return op, nil
}

func (s *Scraper) findDuplicate(ctx context.Context, hash []byte, remotePath string, dest string) (string, error) {
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

func (s *Scraper) doGetRequest(ctx context.Context, path string, dest any) error {
	userId := config.Auth.GetString("UserID")
	const maxAttempts = 3
	for i := 0; i < maxAttempts; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		if err != nil {
			return err
		}

		t := time.Now()

		req.Header.Set("accept", "application/json, text/plain, */*")
		req.Header.Set("user-agent", config.Auth.GetString("exhentai.UserAgent"))
		req.Header.Set("app-token", config.DynamicRules.AppToken)
		req.Header.Set("x-bc", config.Auth.GetString("exhentai.BC"))
		req.Header.Set("time", strconv.FormatInt(t.UnixMilli(), 10))
		req.Header.Set("referer", "https://gelbooru.com/")
		req.Header.Set("user-id", userId)

		req.AddCookie(&http.Cookie{
			Name:  "auth_id",
			Value: config.Auth.GetString("UserID"),
		})
		req.AddCookie(&http.Cookie{
			Name:  "sess",
			Value: config.Auth.GetString("SessionToken"),
		})

		sign, err := GenSignHeader(path, &t, userId)
		if err != nil {
			return err
		}
		req.Header.Set("sign", sign)

		res, err := s.client.Do(req)
		if err != nil {
			return err
		}

		if res.StatusCode == 429 && (i < maxAttempts-1) {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond * time.Duration(math.Pow(2, float64(i)))):
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

		err = json.NewDecoder(res.Body).Decode(dest)
		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("too many failed attempts")
}

type infiniteResponse[T any] struct {
	List    []T  `json:"list"`
	HasMore bool `json:"hasMore"`
}

type ScrapeProfile struct {
	ID       string
	SourceID string
}

type ScrapeProfileList []ScrapeProfile

func (l ScrapeProfileList) ToMapIndexedById() map[string]ScrapeProfile {
	res := make(map[string]ScrapeProfile, len(l))
	for _, v := range l {
		res[v.ID] = v
	}
	return res
}

func (l ScrapeProfileList) ToMapIndexedBySourceId() map[string]ScrapeProfile {
	res := make(map[string]ScrapeProfile, len(l))
	for _, v := range l {
		res[v.SourceID] = v
	}
	return res
}
