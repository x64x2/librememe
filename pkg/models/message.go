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

var MessageBase = Base{
	table: "messages_v2",
}

type Message struct {
	ID           string   `db:"message_id" db_is_id:"1"`
	Source       int      `db:"message_source"`
	SourceID     string   `db:"message_source_id"`
	Author       string   `db:"message_author"`
	SourceChatID string   `db:"message_source_chat_id"`
	Date         int64    `db:"message_date"`
	Text         string   `db:"message_text"`
	Media        MediaCol `db:"message_media"`

	tx       *sqlx.Tx
	modified bool
}

func NewMessage(source int, sourceID string) Message {
	id := uuid.Must(uuid.NewRandom())
	return Message{
		ID:       id.String(),
		Source:   source,
		SourceID: sourceID,
	}
}

func LoadMessage(ctx context.Context, messageID string) (*Message, error) {
	conn := db.Get()
	return loadMessage(ctx, conn, messageID)
}

func LoadMessageForUpdate(ctx context.Context, messageID string) (*Message, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	p, err := loadMessage(ctx, tx, messageID)
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

func FindMessage(ctx context.Context, source int, sourceID string) (*Message, error) {
	conn := db.Get()
	return findMessage(ctx, conn, source, sourceID)
}

func FindMessageForUpdate(ctx context.Context, source int, sourceID string) (*Message, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	p, err := findMessage(ctx, tx, source, sourceID)
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

func loadMessage(ctx context.Context, conn conn, messageID string) (*Message, error) {
	return queryMessage(ctx, conn, `message_id = `+db.GetPlaceholder(), messageID)
}

func findMessage(ctx context.Context, conn conn, source int, sourceID string) (*Message, error) {
	placeholders := db.GetPlaceholders(2)
	return queryMessage(ctx, conn,
		`message_source = `+placeholders[0]+` AND message_source_id = `+placeholders[1],
		source, sourceID,
	)
}

func queryMessage(ctx context.Context, conn conn, where string, params ...any) (*Message, error) {
	q := `
SELECT
	message_id, message_source, message_source_id,
	message_source_chat_id, message_author, message_date, message_text, message_media
FROM ` + MessageBase.TableName() + `
WHERE ` + where

	p := &Message{}

	row := conn.QueryRowContext(ctx, q, params...)
	err := row.Scan(
		&p.ID, &p.Source, &p.SourceID,
		&p.SourceChatID, &p.Author, &p.Date, &p.Text, &p.Media,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading row: %w", err)
	}

	return p, nil
}

func (p *Message) CloseTx() {
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

func (p *Message) GetTx() *sqlx.Tx {
	return p.tx
}

func (p *Message) SetModified() {
	p.modified = true
}

func (m Message) Valid() error {
	if !IsValidID(m.ID) {
		return errors.New("message ID is empty")
	}
	if !db.IsValidSource(m.Source) {
		return errors.New("message source is invalid")
	}
	if m.SourceID == "" {
		return errors.New("message source ID is empty")
	}
	if !IsValidID(m.Author) {
		return errors.New("message author is empty")
	}
	if m.SourceChatID == "" {
		return errors.New("message source chat ID is empty")
	}
	if m.Date <= 0 {
		return errors.New("message date is empty")
	}
	return nil
}

func (p *Message) Save(ctx context.Context) error {
	if p.modified {
		var conn conn
		if p.tx == nil {
			conn = db.Get()
		} else {
			conn = p.tx
		}

		_, err := MessageBase.Save(ctx, conn, p, &SaveOpts{
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
			return fmt.Errorf("failed to commit transaction for message %v: %w", p.ID, err)
		}
	}

	p.tx = nil
	p.modified = false
	return nil
}

func (p Message) String() string {
	enc, _ := json.Marshal(p)
	var res strings.Builder
	res.WriteString("Message: ")
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
