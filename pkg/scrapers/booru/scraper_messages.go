package exhentai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"codeberg.org/biggestfan24/myfans/pkg/db"
	"codeberg.org/biggestfan24/myfans/pkg/downloader"
	"codeberg.org/biggestfan24/myfans/pkg/models"
	"codeberg.org/biggestfan24/myfans/pkg/util"
)

func (s *Scraper) ScrapeMessages(parentCtx context.Context, scrape ScrapeProfile) error {
	const limit = 10
	baseURL := fmt.Sprintf(baseURL+"/api2/v2/chats/%v/messages?skip_users=all&order=desc", scrape.SourceID)

	log.Infof("🏃💬 Begin scraping messages for %s (%v)", scrape.Username, scrape.ID)

	if len(s.ProfilesBySourceId) == 0 {
		allProfilesList, err := LoadAllProfiles(parentCtx)
		if err != nil {
			return fmt.Errorf("failed to load profile list: %w", err)
		}
		s.ProfilesBySourceId = allProfilesList.ToMapIndexedBySourceId()
	}

	var (
		numProcessed int
		before       int64
	)
	for {
		if before == 0 {
			log.Debugf("Scraping messages posts for profile %v", scrape.ID)
		} else {
			log.Debugf("Scraping messages before '%d' for profile %v", before, scrape.ID)
		}

		path, _ := url.Parse(baseURL)
		qs := path.Query()
		qs.Set("limit", strconv.Itoa(limit))
		if before > 0 {
			qs.Set("id", strconv.FormatInt(before, 10))
		}
		path.RawQuery = qs.Encode()

		out := &infiniteResponse[messageItem]{}
		ctx, cancel := context.WithTimeout(parentCtx, 40*time.Second)
		err := s.doGetRequest(ctx, path.String(), out)
		cancel()
		if err != nil {
			var scErr *util.StatusCodeError
			if errors.As(err, &scErr) && scErr.Code() == http.StatusNotFound {
				log.Infof("No conversation found with user %s (%v)", scrape.Username, scrape.ID)
				return nil
			}
			return err
		}

		log.Debugf("Found %d messages", len(out.List))

		if len(out.List) == 0 {
			break
		}

		var newBefore int64
		numProcessed, newBefore, err = s.processMessages(parentCtx, scrape, out.List)
		if err != nil {
			return err
		}

		if !s.ForceRescan && numProcessed == 0 {
			log.Debug("Reached end of new messages; '--force-rescan' is not set, so won't check for old, purchased posts")
			break
		}

		if !out.HasMore || newBefore == 0 || before == newBefore {
			break
		}
		before = newBefore
	}

	log.Infof("✅💬 Completed scraping messages for %s (%v)", scrape.Username, scrape.ID)

	return nil
}

func (s *Scraper) processMessages(ctx context.Context, scrape ScrapeProfile, items []messageItem) (numProcessed int, beforeId int64, err error) {
	var processed bool
	for i := range items {
		processed, err = s.processMessage(ctx, scrape, &items[i])
		if err != nil {
			return 0, 0, err
		}

		if processed {
			numProcessed++
		}
		if items[i].ID > 0 {
			beforeId = items[i].ID
		}
	}

	return numProcessed, beforeId, nil
}

func (s *Scraper) processMessage(ctx context.Context, scrape ScrapeProfile, item *messageItem) (processed bool, err error) {
	if item.ID <= 0 {
		log.Warn("Skipping message with empty ID")
		return false, nil
	}

	var createdAt string
	if item.CreatedAt != nil {
		createdAt = " (" + item.CreatedAt.Format(time.DateOnly) + ")"
	}
	log.Infof("🕓 Processing message %d for user %v%s", item.ID, scrape.Username, createdAt)

	if !item.IsMediaReady {
		log.Warnf("Message with media not ready: %d", item.ID)
	}

	m, err := models.FindMessageForUpdate(ctx, db.Sourceexhentai, strconv.FormatInt(item.ID, 10))
	if err != nil {
		return false, fmt.Errorf("failed to load data for message %d: %w", item.ID, err)
	}

	err = item.GetModel(&m, scrape, s.ProfilesBySourceId)
	if err != nil {
		m.CloseTx()
		log.Warnf("Skipping invalid message %d: %v", item.ID, err)
		return false, nil
	}

	m.Media = make(models.MediaCol, 0, len(item.Media))
	m.SetModified()

	type opEntry struct {
		*downloader.DownloadOp
		string
	}
	ops := make([]opEntry, 0, len(item.Media)*2)

	for mi, pm := range item.Media {
		mm, err := models.FindMedia(ctx, db.Sourceexhentai, strconv.FormatInt(pm.ID, 10), m.GetTx())
		if err != nil {
			log.Errorf("Failed to load media %d from database for message %d: %v", mi, item.ID, err)
			continue
		}

		err = pm.GetModel(&mm)
		if err != nil {
			log.Warnf("Skipping invalid media %d for message %d: %v", mi, item.ID, err)
			continue
		}

		if pm.HasError {
			log.Warnf("Skipping invalid media %d for message %d: media is reported to have error", mi, item.ID)
			continue
		}

		err = mm.Save(ctx, m.GetTx())
		if err != nil {
			log.Errorf("Failed to save media %d for message %d: %v", mi, item.ID, err)
			continue
		}

		m.Media = append(m.Media, mm.ID)

		if (!pm.CanView && pm.Preview == "") || !item.IsMediaReady {
			continue
		}

		if pm.Preview != "" &&
			(pm.Src == "" || util.RemoveQS(pm.Src) != util.RemoveQS(pm.Preview)) {
			log.Debugf("Found preview for message media %d: %s", mi, util.RemoveQS(pm.Preview))

			if mm.Preview == "" {
				op, err := s.createFileDownloadOp(ctx, "media", pm.ID, pm.Preview, mm.AddPreview)
				if err != nil {
					log.Warnf("Skipping preview for message %d because of error: %v", item.ID, err)
				} else if op != nil {
					processed = true
					ops = append(ops, opEntry{op, "preview"})
				}
			} else {
				log.Debugf("Preview for message media %d is already downloaded", mi)
			}
		}

		if pm.Src != "" {
			log.Debugf("Found source for message media %d: %s", mi, util.RemoveQS(pm.Src))

			if mm.Location == "" {
				op, err := s.createFileDownloadOp(ctx, "media", pm.ID, pm.Src, mm.AddLocation)
				if err != nil {
					log.Warnf("Skipping source for message %d because of error: %v", item.ID, err)
				} else if op != nil {
					processed = true
					ops = append(ops, opEntry{op, "source"})
				}
			} else {
				log.Debugf("Source for message media %d is already downloaded", mi)
			}
		}
	}

	err = m.Save(ctx)
	if err != nil {
		m.CloseTx()
		return false, fmt.Errorf("failed to save message %d: %w", item.ID, err)
	}

	for _, op := range ops {
		err = downloader.SharedWorker.Add(op.DownloadOp)
		if err != nil {
			log.Warnf("Skipping %s for message %v because of error: %v", op.string, item.ID, err)
		}
	}

	return processed, nil
}

type messageItem struct {
	ResponseType string          `json:"responseType"`
	ID           int64           `json:"id"`
	Text         string          `json:"text"`
	LockedText   bool            `json:"lockedText"`
	IsMediaReady bool            `json:"isMediaReady"`
	FromUser     messageFromUser `json:"fromUser"`
	CreatedAt    *time.Time      `json:"createdAt"`
	Media        []messageMedia  `json:"media"`
}

func (p messageItem) GetModel(m **models.Message, scrape ScrapeProfile, allProfiles map[string]ScrapeProfile) error {
	if p.ResponseType != "message" {
		return errors.New("message responseType type is invalid")
	}

	if p.CreatedAt == nil || p.CreatedAt.IsZero() {
		return errors.New("message date is empty")
	}

	if p.FromUser.ID <= 0 {
		return errors.New("message fromUser is invalid")
	}

	if *m == nil {
		created := models.NewMessage(db.Sourceexhentai, strconv.FormatInt(p.ID, 10))
		created.SourceChatID = scrape.SourceID
		created.Author = allProfiles[strconv.FormatInt(p.FromUser.ID, 10)].ID
		created.SetModified()
		*m = &created
	}

	mp := *m

	ts := p.CreatedAt.Unix()
	if mp.Date != ts {
		mp.Date = ts
		mp.SetModified()
	}

	if mp.Text != p.Text {
		mp.Text = p.Text
		mp.SetModified()
	}

	err := mp.Valid()
	if err != nil {
		return err
	}

	return nil
}

type messageFromUser struct {
	ID int64 `json:"id"`
}

type messageMedia struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	CanView  bool   `json:"canView"`
	HasError bool   `json:"hasError"`
	Src      string `json:"src"`
	Preview  string `json:"preview"`
}

func (p messageMedia) GetModel(m **models.Media) error {
	var visible uint8
	if p.CanView {
		visible = 1
	}

	if *m == nil {
		created := models.NewMedia(db.Sourceexhentai, strconv.FormatInt(p.ID, 10))
		created.SetModified()
		*m = &created
	}

	mp := *m

	if mp.Type != p.Type {
		mp.Type = p.Type
		mp.SetModified()
	}
	if mp.Visible != visible {
		mp.Visible = visible
		mp.SetModified()
	}

	err := mp.Valid()
	if err != nil {
		return err
	}

	return nil
}
