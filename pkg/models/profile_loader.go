package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"codeberg.org/biggestfan24/myfans/pkg/db"
	"github.com/google/uuid"
)

func LoadProfileIDs(ctx context.Context, profiles []string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)

	switch db.Get().DriverName() {
	case "pgx":
		rows, err = loadProfileIDsPgx(ctx, profiles, db.Sourceexhentai)
	case "sqlite":
		rows, err = loadProfileIDsSqlite(ctx, profiles, db.Sourceexhentai)
	default:
		return nil, errors.New("invalid DB driver")
	}

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	resIDs := make([]string, len(profiles))
	var (
		n  int
		id string
	)
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if IsValidID(id) {
			resIDs[n] = id
			n++
		}
	}

	return resIDs[:n], nil
}

func loadProfileIDsSqlite(ctx context.Context, profiles []string, source int) (*sql.Rows, error) {
	params := make([]any, (len(profiles)*3)+1)
	placeholders := make([]string, len(profiles))
	for i, p := range profiles {
		params[i] = p
		params[i+len(profiles)] = p
		params[i+(len(profiles)*2)] = p
		placeholders[i] = "?"
	}
	params[len(params)-1] = source

	q := fmt.Sprintf(`SELECT profile_id FROM `+ProfileBase.TableName()+` WHERE (profile_username IN (%[1]s) OR profile_source_id IN (%[1]s) OR profile_id IN (%[1]s)) AND profile_source = ?`, strings.Join(placeholders, ", "))
	rows, err := db.Get().QueryContext(ctx, q, params...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to load profile IDs: %w", err)
	}

	return rows, nil
}

func loadProfileIDsPgx(ctx context.Context, profiles []string, source int) (*sql.Rows, error) {
	profilesUUID := make([]uuid.UUID, 0, len(profiles))
	for _, p := range profiles {
		u, err := uuid.Parse(p)
		if err == nil {
			profilesUUID = append(profilesUUID, u)
		}
	}

	rows, err := db.Get().QueryContext(ctx, `SELECT profile_id FROM `+ProfileBase.TableName()+` WHERE (profile_username = any($1) OR profile_source_id = any($1) OR profile_id = any($2)) AND profile_source = $3`, profiles, profilesUUID, source)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to load profile IDs: %w", err)
	}

	return rows, nil
}
