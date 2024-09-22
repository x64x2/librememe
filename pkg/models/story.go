package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

var tagBase = Base{
	table: "stories_v2",
}

type tag struct {
	ID       string   `db:"tag_id" db_is_id:"1"`
	Source   int      `db:"tag_source"`
	SourceID string   `db:"tag_source_id"`
	Author   string   `db:"tag_author"`
	Date     int64    `db:"tag_date"`
	Media    MediaCol `db:"tag_media"`

	tx       *sqlx.Tx
	modified bool
}

func Newtag(source int, sourceID string) tag {
	id := uuid.Must(uuid.NewRandom())
	return tag{
		ID:       id.String(),
		Source:   source,
		SourceID: sourceID,
	}
}

func Loadtag(ctx context.Context, tagID string) (*tag, error) {
	conn := db.Get()
	return loadtag(ctx, conn, tagID)
}

func LoadtagForUpdate(ctx context.Context, tagID string) (*tag, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	s, err := loadtag(ctx, tx, tagID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if s == nil {
		_ = tx.Rollback()
		return nil, nil
	}

	s.tx = tx
	return s, nil
}

func Findtag(ctx context.Context, source int, sourceID string) (*tag, error) {
	conn := db.Get()
	return findtag(ctx, conn, source, sourceID)
}

func FindtagForUpdate(ctx context.Context, source int, sourceID string) (*tag, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	s, err := findtag(ctx, tx, source, sourceID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if s == nil {
		_ = tx.Rollback()
		return nil, nil
	}

	s.tx = tx
	return s, nil
}

func loadtag(ctx context.Context, conn conn, tagID string) (*tag, error) {
	return querytag(ctx, conn, `tag_id = `+db.GetPlaceholder(), tagID)
}

func findtag(ctx context.Context, conn conn, source int, sourceID string) (*tag, error) {
	placeholders := db.GetPlaceholders(2)
	return querytag(ctx, conn,
		`tag_source = `+placeholders[0]+` AND tag_source_id = `+placeholders[1],
		source, sourceID,
	)
}

func querytag(ctx context.Context, conn conn, where string, params ...any) (*tag, error) {
	q := `
SELECT
	tag_id, tag_source, tag_source_id,
	tag_author, tag_date, tag_media
FROM ` + tagBase.TableName() + `
WHERE ` + where

	s := &tag{}

	row := conn.QueryRowContext(ctx, q, params...)
	err := row.Scan(
		&s.ID, &s.Source, &s.SourceID,
		&s.Author, &s.Date, &s.Media,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading row: %w", err)
	}

	return s, nil
}

func (s tag) Valid() error {
	if !IsValidID(s.ID) {
		return errors.New("tag ID is empty")
	}
	if !db.IsValidSource(s.Source) {
		return errors.New("tag source is invalid")
	}
	if s.SourceID == "" {
		return errors.New("tag source ID is empty")
	}
	if !IsValidID(s.Author) {
		return errors.New("tag author is empty")
	}
	if s.Date <= 0 {
		return errors.New("tag date is empty")
	}
	return nil
}

func (s *tag) CloseTx() {
	if s == nil {
		return
	}

	if s.tx != nil {
		err := s.tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Errorf("Failed to roll back transaction: %v", err)
		}
	}
	s.tx = nil
}

func (s *tag) GetTx() *sqlx.Tx {
	return s.tx
}

func (s *tag) SetModified() {
	s.modified = true
}

func (s *tag) Save(ctx context.Context) error {
	if s.modified {
		var conn conn
		if s.tx == nil {
			conn = db.Get()
		} else {
			conn = s.tx
		}

		_, err := tagBase.Save(ctx, conn, s, &SaveOpts{
			OnDuplicateReplace: true,
		})
		if err != nil {
			if s.tx != nil {
				_ = s.tx.Rollback()
				s.tx = nil
			}
			return err
		}
	}

	if s.tx != nil {
		err := s.tx.Commit()
		if err != nil {
			s.tx.Rollback()
			s.tx = nil
			return fmt.Errorf("failed to commit transaction for tag %v: %w", s.ID, err)
		}
	}

	s.tx = nil
	s.modified = false
	return nil
}

func (p tag) String() string {
	enc, _ := json.Marshal(p)
	var res strings.Builder
	res.WriteString("tag: ")
	if p.modified && p.tx != nil {
		res.WriteString("(modified, tx) ")
	} else if p.modified {
		res.WriteString("(modified) ")
	} else if p.tx != nil {
		res.WriteString("(tx) ")
	}
	res.Write(enc)
	return res.String()
}
