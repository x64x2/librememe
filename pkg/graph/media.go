package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func fetchMedia(ctx context.Context, media []*model.Media) ([]*model.Media, error) {
	conn := db.Get()

	var (
		fragment string
		params   []any
	)
	switch conn.DriverName() {
	case "sqlite":
		in := strings.Repeat("?,", len(media))
		in = in[:len(in)-1]
		fragment = "IN(" + in + ")"
		params = make([]any, len(media))
		for i, m := range media {
			params[i] = m.ID.String()
		}
	case "pgx":
		fragment = "= any($1)"
		mediaId := make([]uuid.UUID, len(media))
		for i, m := range media {
			mediaId[i] = m.ID
		}
		params = []any{mediaId}
	}

	q := `SELECT
	media_id, media_source, media_source_id,
	media_type, media_visible, media_location, media_preview
	FROM ` + models.MediaBase.TableName() + `
	WHERE media_id ` + fragment

	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := conn.QueryContext(queryCtx, q, params...)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	res := map[uuid.UUID]*model.Media{}
	for rows.Next() {
		m := &model.Media{}
		var (
			visible           bool
			location, preview string
		)
		err = rows.Scan(
			&m.ID, &m.Source, &m.SourceID,
			&m.Type, &visible, &location, &preview,
		)
		if err != nil {
			return nil, fmt.Errorf("error reading row: %w", err)
		}

		m.Visible = &visible
		if location != "" {
			m.Location = &location
		}
		if preview != "" {
			m.Preview = &preview
		}
		res[m.ID] = m
	}

	for i, m := range media {
		if res[m.ID] != nil {
			media[i] = res[m.ID]
		}
	}

	return media, nil
}
