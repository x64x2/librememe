package exhentai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

)

const (
	exhentaiBaseUrl      = "https://exhentai.org"
	exhentaiDataBaseUrl  = BaseUrl + "/data"
	exhentaiApiBaseUrl   = exhentaiBaseUrl + "/api/v1"
	exhentaiIconsBaseUrl = "https://img.exhentai.org/icons"
)

type exhentaiImporter struct {
	service     string
	source      int
	forceRescan bool
	client      *http.Client
}

func NewexhentaiImporter(service string, forceRescan bool) *exhentaiImporter {
	var source int
	switch service {
	case "exhentai":
		source = db.Sourceexhentai
	case "fansly":
		source = db.SourceexhentaiFansly
	default:
		panic("invalid service name: " + service)
	}

	return &exhentaiImporter{
		service:     service,
		source:      source,
		forceRescan: forceRescan,
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

func (c exhentaiImporter) ImportProfile(ctx context.Context, profileName string) error {
	profile, err := c.loadProfile(ctx, profileName)
	if err != nil {
		return fmt.Errorf("failed to load profile %s: %w", profileName, err)
	}

	err = c.loadPosts(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to load posts: %w", err)
	}

	return nil
}

func (c exhentaiImporter) loadProfile(ctx context.Context, profileName string) (*models.Profile, error) {
	u := fmt.Sprintf("%s/%s/user/%s/profile", exhentaiApiBaseUrl, c.service, profileName)

	out := exhentaiProfile{}
	err := c.doGetJSONRequest(ctx, u, &out)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if out.ID == "" || out.Name == "" || out.Service != c.service || out.Updated.IsZero() {
		return nil, errors.New("invalid profile: missing required fields")
	}

	mm, err := models.FindProfileForUpdate(ctx, c.source, out.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load data for profile: %w", err)
	}

	err = out.GetModel(&mm, c.source)
	if err != nil {
		mm.CloseTx()
		return nil, err
	}

	err = mm.Save(ctx)
	if err != nil {
		mm.CloseTx()
		return nil, fmt.Errorf("failed to save profile: %w", err)
	}

	return mm, nil
}

func (c exhentaiImporter) loadPosts(ctx context.Context, profile *models.Profile) error {
	const pagination = 50
	for offset := 0; ; offset += pagination {
		log.Debugf("Requesting posts with offset %d", offset)

		u := fmt.Sprintf("%s/%s/user/%s?o=%d", exhentaiApiBaseUrl, c.service, profile.SourceID, offset)

		out := []exhentaiPost{}
		err := c.doGetJSONRequest(ctx, u, &out)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		log.Debugf("Found %d posts", len(out))

		var processed bool
		for i, post := range out {
			pp, err := c.processPost(ctx, &post, profile)
			if err != nil {
				return fmt.Errorf("failed to process post %d: %w", i, err)
			}
			processed = processed || pp
		}

		if len(out) < pagination {
			break
		}

		if !c.forceRescan && !processed {
			log.Debug("Reached end of new posts; '--force-rescan' is not set, so won't check for old, purchased posts")
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (c exhentaiImporter) processPost(ctx context.Context, item *exhentaiPost, profile *models.Profile) (bool, error) {
	log.Infof("📝 Processing post %v for user %v (%v)", item.ID, profile.Username, item.Published.Format(time.DateOnly))

	var invisibleMedia int

	m, err := models.FindPostForUpdate(ctx, profile.Source, item.ID)
	if err != nil {
		return false, fmt.Errorf("failed to load data for post %v: %w", item.ID, err)
	}

	if m == nil || item.GetFilesCount() != len(m.Media) {
		invisibleMedia, err = c.findInvisibleMedia(ctx, profile, item.ID)
		if err != nil {
			log.Warnf("Failed to look for invisible media for post %v: %v", item.ID, err)
		}
	}

	err = item.GetModel(&m, profile)
	if err != nil {
		m.CloseTx()
		log.Warnf("Skipping invalid post %v: %v", item.ID, err)
		return false, nil
	}

	files := item.GetFiles()
	newMedia := make(models.MediaCol, 0, len(files)+invisibleMedia)

	ops := make([]*downloader.DownloadOp, 0, len(files))
	for mi, pm := range files {
		mm, err := models.FindMedia(ctx, profile.Source, mediaID(m, mi), m.GetTx())
		if err != nil {
			log.Errorf("Failed to load media %d from database for post %s: %v", mi, item.ID, err)
			continue
		}

		err = pm.GetModel(&mm, m, mi)
		if err != nil {
			log.Warnf("Skipping invalid media %d for post %s: %v", mi, item.ID, err)
			continue
		}

		err = mm.Save(ctx, m.GetTx())
		if err != nil {
			log.Errorf("Failed to save media %d for post %s: %v", mi, item.ID, err)
			continue
		}

		newMedia = append(newMedia, mm.ID)

		if pm.Path != "" {
			log.Debugf("Found source for post media %d: %s", mi, pm.Path)

			if mm.Location == "" {
				op, err := c.createFileDownloadOp(ctx, "media", pm.Path, pm.Name, mm.AddLocation)
				if err != nil {
					log.Warnf("Skipping source for post %v because of error: %v", item.ID, err)
				} else if op != nil {
					ops = append(ops, op)
				}
			} else {
				log.Debugf("Source for post media %d is already downloaded", mi)
			}
		}
	}

	if invisibleMedia > 0 {
		log.Debugf("Post %v contains %d invisible media", item.ID, invisibleMedia)
	}

	mediaCount := len(newMedia)
	for i := 0; i < invisibleMedia; i++ {
		mid := mediaID(m, mediaCount+i)
		mm, err := models.FindMedia(ctx, profile.Source, mid, m.GetTx())
		if err != nil {
			log.Errorf("Failed to load invisible media %d from database for post %s: %v", i, item.ID, err)
			continue
		}

		if mm == nil {
			created := models.NewMedia(m.Source, mid)
			mm = &created
		}

		mm.Visible = 0
		mm.Type = "photo"
		mm.SetModified()

		err = mm.Save(ctx, m.GetTx())
		if err != nil {
			log.Errorf("Failed to save invisible media %d for post %s: %v", i, item.ID, err)
			continue
		}

		newMedia = append(newMedia, mm.ID)
	}

	if !m.Media.Equal(newMedia) {
		m.Media = newMedia
		m.SetModified()
	}

	modified := m.IsModified()

	err = m.Save(ctx)
	if err != nil {
		m.CloseTx()
		return false, fmt.Errorf("failed to save post %v: %w", item.ID, err)
	}

	for _, op := range ops {
		err = downloader.SharedWorker.Add(op)
		if err != nil {
			log.Warnf("Skipping source for post %v because of error: %v", item.ID, err)
		}
	}

	return modified, nil
}

var invisibleMediaRegex = regexp.MustCompile(`<pre>\s*This post is missing paid rewards from a higher tier or payment.\s*(\d+) media(?:.*?)</pre>`)

func (c exhentaiImporter) findInvisibleMedia(parentCtx context.Context, profile *models.Profile, postID string) (int, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 40*time.Second)
	defer cancel()

	var count int

	u := fmt.Sprintf("%s/%s/user/%s/post/%s", exhentaiBaseUrl, c.service, profile.SourceID, postID)
	err := c.doGetRequest(ctx, u,
		func(h http.Header) {
			h.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
		},
		func(r io.ReadCloser) error {
			body, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			match := invisibleMediaRegex.FindSubmatch(body)
			if len(match) != 2 {
				return nil
			}
			count, _ = strconv.Atoi(string(match[1]))
			return nil
		},
	)
	if err != nil {
		return 0, err
	}

	return count, nil
}
