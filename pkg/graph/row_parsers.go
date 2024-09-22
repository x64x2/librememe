package graph

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type rower interface{ Scan(dest ...any) error }

const (
	profileCols = "profile_id, profile_source, profile_source_id, profile_username, profile_name, profile_avatar_location, profile_header_location"
	postCols    = "post_id, post_source, post_source_id, post_author, post_date, post_text, post_media"
)

func parseProfileRow(row rower) (*model.Profile, error) {
	var avatar, header string
	p := &model.Profile{}
	err := row.Scan(&p.ID, &p.Source, &p.SourceID, &p.Username, &p.Name, &avatar, &header)
	if err != nil {
		return nil, fmt.Errorf("failed to scan profile row: %w", err)
	}

	if avatar != "" {
		p.Avatar = &avatar
	}
	if header != "" {
		p.Header = &header
	}

	return p, nil
}

func parsePostRow(row rower) (*model.Post, error) {
	p := &model.Post{}
	var (
		author uuid.UUID
		date   int64
		media  models.MediaCol
	)
	err := row.Scan(&p.ID, &p.Source, &p.SourceID, &author, &date, &p.Text, &media)
	if err != nil {
		return nil, fmt.Errorf("failed to scan post row: %w", err)
	}

	if date > 0 {
		p.Date = time.Unix(date, 0)
	}
	if IsValidUUID(author) {
		p.Author = &model.Profile{
			ID: author,
		}
	}
	if len(media) > 0 {
		p.Media = make([]*model.Media, len(media))
		for i, v := range media {
			u, err := uuid.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse media ID '%v': %w", v, err)
			}
			if !IsValidUUID(u) {
				return nil, fmt.Errorf("invalid media ID: %v", v)
			}
			p.Media[i] = &model.Media{
				ID: u,
			}
		}
	}

	return p, nil
}

func parseMessageRow(row rower) (*model.Message, error) {
	m := &model.Message{}
	var (
		author uuid.UUID
		date   int64
		media  models.MediaCol
	)
	err := row.Scan(&m.ID, &m.Source, &m.SourceID, &author, &date, &m.Text, &media)
	if err != nil {
		return nil, fmt.Errorf("failed to scan message row: %w", err)
	}

	if date > 0 {
		m.Date = time.Unix(date, 0)
	}
	if IsValidUUID(author) {
		m.Author = &model.Profile{
			ID: author,
		}
	}
	if len(media) > 0 {
		m.Media = make([]*model.Media, len(media))
		for i, v := range media {
			u, err := uuid.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse media ID '%v': %w", v, err)
			}
			if !IsValidUUID(u) {
				return nil, fmt.Errorf("invalid media ID: %v", v)
			}
			m.Media[i] = &model.Media{
				ID: u,
			}
		}
	}

	return m, nil
}

func parsetagRow(row rower) (*model.tag, error) {
	s := &model.tag{}
	var (
		author uuid.UUID
		date   int64
		media  models.MediaCol
	)
	err := row.Scan(&s.ID, &s.Source, &s.SourceID, &author, &date, &media)
	if err != nil {
		return nil, fmt.Errorf("failed to scan tag row: %w", err)
	}

	if date > 0 {
		s.Date = time.Unix(date, 0)
	}
	if IsValidUUID(author) {
		s.Author = &model.Profile{
			ID: author,
		}
	}
	if len(media) > 0 {
		s.Media = make([]*model.Media, len(media))
		for i, v := range media {
			u, err := uuid.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse media ID '%v': %w", v, err)
			}
			if !IsValidUUID(u) {
				return nil, fmt.Errorf("invalid media ID: %v", v)
			}
			s.Media[i] = &model.Media{
				ID: u,
			}
		}
	}

	return s, nil
}
