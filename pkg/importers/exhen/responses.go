package exhentai

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type exhentaiProfile struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Service string    `json:"service"`
	Updated time.Time `json:"updated"`
}

func (p exhentaiProfile) GetModel(m **models.Profile, source int) error {
	if *m == nil {
		created := models.NewProfile(source, p.ID)
		created.SetModified()
		*m = &created
	}

	mp := *m

	if mp.Username != p.Name {
		mp.Username = p.Name
		mp.SetModified()
	}
	if mp.Name != p.Name {
		mp.Name = p.Name
		mp.SetModified()
	}

	ts := p.Updated.Unix()
	if mp.LastScrape != ts {
		mp.LastScrape = ts
		mp.SetModified()
	}

	err := mp.Valid()
	if err != nil {
		return err
	}

	return nil
}

func (p *exhentaiProfile) UnmarshalJSON(b []byte) error {
	type alias exhentaiProfile
	wrapper := struct {
		*alias
		Updated string `json:"updated"`
	}{
		alias: (*alias)(p),
	}
	err := json.Unmarshal(b, &wrapper)
	if err != nil {
		return err
	}

	p.Updated, err = time.ParseInLocation("2006-01-02T15:04:05", wrapper.Updated, time.UTC)
	if err != nil {
		return fmt.Errorf("invalid updated time '%s': %w", wrapper.Updated, err)
	}

	return nil
}

type exhentaiPost struct {
	ID          string       `json:"id"`
	Content     string       `json:"content"`
	Published   time.Time    `json:"published"`
	File        exhentaiFile   `json:"file"`
	Attachments []exhentaiFile `json:"attachments"`
}

func (p *exhentaiPost) UnmarshalJSON(b []byte) error {
	type alias exhentaiPost
	wrapper := struct {
		*alias
		Published string `json:"published"`
	}{
		alias: (*alias)(p),
	}
	err := json.Unmarshal(b, &wrapper)
	if err != nil {
		return err
	}

	p.Published, err = time.ParseInLocation("2006-01-02T15:04:05", wrapper.Published, time.UTC)
	if err != nil {
		return fmt.Errorf("invalid published time '%s': %w", wrapper.Published, err)
	}

	return nil
}

func (p exhentaiPost) GetModel(m **models.Post, profile *models.Profile) error {
	if p.Published.IsZero() {
		return errors.New("post published date is empty")
	}

	if *m == nil {
		created := models.NewPost(profile.Source, p.ID)
		created.Author = profile.ID
		created.SetModified()
		*m = &created
	}

	mp := *m

	ts := p.Published.Unix()
	if mp.Date != ts {
		mp.Date = ts
		mp.SetModified()
	}
	if mp.Text != p.Content {
		mp.Text = p.Content
		mp.SetModified()
	}

	err := mp.Valid()
	if err != nil {
		return err
	}

	return nil
}

func (p exhentaiPost) GetFilesCount() int {
	if p.File.Name == "" || p.File.Path == "" {
		return len(p.Attachments)
	}

	return len(p.Attachments) + 1
}

func (p exhentaiPost) GetFiles() []exhentaiFile {
	if p.File.Name == "" || p.File.Path == "" {
		return p.Attachments
	}

	result := make([]exhentaiFile, len(p.Attachments)+1)
	result[0] = exhentaiFile{
		Name: p.File.Name,
		Path: p.File.Path,
	}
	copy(result[1:], p.Attachments)
	return result
}

type exhentaiFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (f exhentaiFile) GetModel(m **models.Media, post *models.Post, idx int) error {
	if *m == nil {
		created := models.NewMedia(post.Source, mediaID(post, idx))
		created.SetModified()
		*m = &created
	}

	mp := *m

	ft, err := f.GetType()
	if err != nil {
		return fmt.Errorf("failed to determine file type: %w", err)
	}
	if mp.Type != ft {
		mp.Type = ft
		mp.SetModified()
	}
	if mp.Visible != 1 {
		mp.Visible = 1
		mp.SetModified()
	}

	err = mp.Valid()
	if err != nil {
		return err
	}

	return nil
}

func (f exhentaiFile) GetType() (string, error) {
	return util.GetFileType(f.Name)
}
