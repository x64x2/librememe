package graph

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UUID = uuid.UUID

func (r *Resolver) profileDataloaderFn(keys []UUID) ([]*model.Profile, []error) {
	conn := db.Get()

	errs := make([]error, len(keys))

	var (
		where  string
		params []any
	)
	switch conn.DriverName() {
	case "sqlite":
		where = "profile_id IN (?" + strings.Repeat(",?", len(keys)-1) + ")"
		params = make([]any, len(keys))
		for i, v := range keys {
			params[i] = v
		}
	case "pgx":
		where = "profile_id = any($1)"
		params = []any{keys}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := conn.QueryContext(ctx, `SELECT
	`+profileCols+`
	FROM `+models.ProfileBase.TableName()+`
	WHERE `+where,
		params...,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		for i := range errs {
			errs[i] = fmt.Errorf("error querying the database: %w", err)
		}
		return nil, errs
	}
	defer res.Close()

	profilesById := map[UUID]*model.Profile{}
	for res.Next() {
		p, err := parseProfileRow(res)
		if err != nil {
			for i := range errs {
				errs[i] = fmt.Errorf("error parsing row: %w", err)
			}
			return nil, errs
		}
		profilesById[p.ID] = p
	}

	profiles := make([]*model.Profile, len(keys))
	for i, v := range keys {
		p, ok := profilesById[v]
		if ok {
			profiles[i] = p
		} else {
			errs[i] = fmt.Errorf("user not found: %v", v)
		}
	}

	return profiles, errs
}
