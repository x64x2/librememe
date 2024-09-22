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
	log "github.com/sirupsen/logrus"

	"codeberg.org/biggestfan24/myfans/pkg/db"
)

var PostBase = Base{
	table: "posts_v2",
}

type Post struct {
	ID       string   `db:"post_id" db_is_id:"1"`
	Source   int      `db:"post_source"`
	SourceID string   `db:"post_source_id"`
	Author   string   `db:"post_author"`
	Date     int64    `db:"post_date"`
	Text     string   `db:"post_text"`
	Media    MediaCol `db:"post_media"`

	tx       *sqlx.Tx
	modified bool
}

func NewPost(source int, sourceID string) Post {
	id := uuid.Must(uuid.NewRandom())
	return Post{
		ID:       id.String(),
		Source:   source,
		SourceID: sourceID,
	}
}

func LoadPost(ctx context.Context, postID string) (*Post, error) {
	conn := db.Get()
	return loadPost(ctx, conn, postID)
}

func LoadPostForUpdate(ctx context.Context, postID string) (*Post, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	p, err := loadPost(ctx, tx, postID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if p == nil {
		_ = tx.Rollback()
		return nil, nil
	}

	p.tx = tx
	return p, nil
}

func FindPost(ctx context.Context, source int, sourceID string) (*Post, error) {
	conn := db.Get()
	return findPost(ctx, conn, source, sourceID)
}

func FindPostForUpdate(ctx context.Context, source int, sourceID string) (*Post, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	p, err := findPost(ctx, tx, source, sourceID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if p == nil {
		_ = tx.Rollback()
		return nil, nil
	}

	p.tx = tx
	return p, nil
}

func loadPost(ctx context.Context, conn conn, postID string) (*Post, error) {
	return queryPost(ctx, conn, `post_id = `+db.GetPlaceholder(), postID)
}

func findPost(ctx context.Context, conn conn, source int, sourceID string) (*Post, error) {
	placeholders := db.GetPlaceholders(2)
	return queryPost(ctx, conn,
		`post_source = `+placeholders[0]+` AND post_source_id = `+placeholders[1],
		source, sourceID,
	)
}

func queryPost(ctx context.Context, conn conn, where string, params ...any) (*Post, error) {
	q := `
SELECT
	post_id, post_source, post_source_id,
	post_author, post_date, post_text, post_media
FROM ` + PostBase.TableName() + `
WHERE ` + where

	p := &Post{}

	row := conn.QueryRowContext(ctx, q, params...)
	err := row.Scan(
		&p.ID, &p.Source, &p.SourceID,
		&p.Author, &p.Date, &p.Text, &p.Media,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading row: %w", err)
	}

	return p, nil
}

func (p *Post) CloseTx() {
	if p == nil {
		return
	}

	if p.tx != nil {
		err := p.tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Errorf("Failed to roll back transaction: %v", err)
		}
	}
	p.tx = nil
}

func (p *Post) GetTx() *sqlx.Tx {
	return p.tx
}

func (p *Post) SetModified() {
	p.modified = true
}

func (p Post) IsModified() bool {
	return p.modified
}

func (p Post) Valid() error {
	if !IsValidID(p.ID) {
		return errors.New("post ID is empty")
	}
	if !db.IsValidSource(p.Source) {
		return errors.New("post source is invalid")
	}
	if p.SourceID == "" {
		return errors.New("post source ID is empty")
	}
	if !IsValidID(p.Author) {
		return errors.New("post author is empty")
	}
	if p.Date <= 0 {
		return errors.New("post date is empty")
	}
	return nil
}

func (p *Post) Save(ctx context.Context) error {
	if p.modified {
		var conn conn
		if p.tx == nil {
			conn = db.Get()
		} else {
			conn = p.tx
		}

		_, err := PostBase.Save(ctx, conn, p, &SaveOpts{
			OnDuplicateReplace: true,
		})
		if err != nil {
			if p.tx != nil {
				_ = p.tx.Rollback()
				p.tx = nil
			}
			return err
		}
	}

	if p.tx != nil {
		err := p.tx.Commit()
		if err != nil {
			p.tx.Rollback()
			p.tx = nil
			return fmt.Errorf("failed to commit transaction for post %v: %w", p.ID, err)
		}
	}

	p.tx = nil
	p.modified = false
	return nil
}

func (p Post) String() string {
	enc, _ := json.Marshal(p)
	var res strings.Builder
	res.WriteString("Post: ")
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
