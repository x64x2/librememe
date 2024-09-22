package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

)

var ProfileBase = Base{
	table: "profiles_v2",
}

type Profile struct {
	ID                     string `db:"profile_id" db_is_id:"1"`
	Source                 int    `db:"profile_source"`
	SourceID               string `db:"profile_source_id"`
	Username               string `db:"profile_username"`
	Name                   string `db:"profile_name"`
	AvatarLocation         string `db:"profile_avatar_location"`
	AvatarOriginalLocation string `db:"profile_avatar_original_location"`
	HeaderLocation         string `db:"profile_header_location"`
	HeaderOriginalLocation string `db:"profile_header_original_location"`
	LastScrape             int64  `db:"profile_last_scrape"`

	tx       *sqlx.Tx
	modified bool
	txLock   *sync.Mutex
}

func NewProfile(source int, sourceID string) Profile {
	id := uuid.Must(uuid.NewRandom())
	return Profile{
		ID:       id.String(),
		Source:   source,
		SourceID: sourceID,
		txLock:   &sync.Mutex{},
	}
}

func LoadProfile(ctx context.Context, profileID string) (*Profile, error) {
	conn := db.Get()
	return loadProfile(ctx, conn, profileID)
}

func LoadProfileForUpdate(ctx context.Context, profileID string) (*Profile, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	p, err := loadProfile(ctx, tx, profileID)
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

func FindProfile(ctx context.Context, source int, sourceID string) (*Profile, error) {
	conn := db.Get()
	return findProfile(ctx, conn, source, sourceID)
}

func FindProfileForUpdate(ctx context.Context, source int, sourceID string) (*Profile, error) {
	conn := db.Get()
	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	p, err := findProfile(ctx, tx, source, sourceID)
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

func loadProfile(ctx context.Context, conn conn, profileID string) (*Profile, error) {
	return queryProfile(ctx, conn, `profile_id = `+db.GetPlaceholder(), profileID)
}

func findProfile(ctx context.Context, conn conn, source int, sourceID string) (*Profile, error) {
	placeholders := db.GetPlaceholders(2)
	return queryProfile(ctx, conn,
		`profile_source = `+placeholders[0]+` AND profile_source_id = `+placeholders[1],
		source, sourceID,
	)
}

func queryProfile(ctx context.Context, conn conn, where string, params ...any) (*Profile, error) {
	q := `
SELECT
	profile_id, profile_source, profile_source_id,
	profile_username, profile_name, 
	profile_avatar_location, profile_avatar_original_location, 
	profile_header_location, profile_header_original_location, 
	profile_last_scrape
FROM ` + ProfileBase.TableName() + `
WHERE ` + where

	p := &Profile{
		txLock: &sync.Mutex{},
	}

	row := conn.QueryRowContext(ctx, q, params...)
	err := row.Scan(
		&p.ID, &p.Source, &p.SourceID,
		&p.Username, &p.Name,
		&p.AvatarLocation, &p.AvatarOriginalLocation,
		&p.HeaderLocation, &p.HeaderOriginalLocation,
		&p.LastScrape,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading row: %w", err)
	}

	return p, nil
}

func (p *Profile) SetModified() {
	p.modified = true
}

func (p Profile) Valid() error {
	if !IsValidID(p.ID) {
		return errors.New("profile ID is empty")
	}
	if !db.IsValidSource(p.Source) {
		return errors.New("profile source is invalid")
	}
	if p.SourceID == "" {
		return errors.New("profile source ID is empty")
	}
	if p.Username == "" {
		return errors.New("profile username is empty")
	}
	return nil
}

func (p *Profile) Save(ctx context.Context) error {
	p.txLock.Lock()
	defer p.txLock.Unlock()

	if p.modified {
		var conn conn
		if p.tx == nil {
			conn = db.Get()
		} else {
			conn = p.tx
		}

		_, err := ProfileBase.Save(ctx, conn, p, &SaveOpts{
			OnDuplicateIgnore: true,
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
			return fmt.Errorf("failed to commit transaction for profile %v: %w", p.ID, err)
		}
	}

	p.tx = nil
	p.modified = false
	return nil
}

func (p *Profile) CloseTx() {
	if p == nil {
		return
	}

	p.txLock.Lock()
	defer p.txLock.Unlock()

	if p.tx != nil {
		err := p.tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Errorf("Failed to roll back transaction: %v", err)
		}
	}
	p.tx = nil
}

func (p *Profile) AddAvatar(ctx context.Context, location string, originalLocation string) error {
	p.AvatarLocation = location
	p.AvatarOriginalLocation = originalLocation
	return p.doUpdateFields(ctx, []string{"profile_avatar_location", "profile_avatar_original_location"})
}

func (p *Profile) AddHeader(ctx context.Context, location string, originalLocation string) error {
	p.HeaderLocation = location
	p.HeaderOriginalLocation = originalLocation
	return p.doUpdateFields(ctx, []string{"profile_header_location", "profile_header_original_location"})
}

func (p *Profile) doUpdateFields(ctx context.Context, updateCols []string) (err error) {
	p.txLock.Lock()
	defer p.txLock.Unlock()

	if p.tx == nil {
		p.tx, err = db.Get().BeginTxx(ctx, nil)
		if err != nil {
			if p.tx != nil {
				p.tx.Rollback()
				p.tx = nil
			}
			return fmt.Errorf("failed to begin transaction for profile %v: %w", p.ID, err)
		}
	}

	err = ProfileBase.Update(ctx, p.tx, p, updateCols)
	if err != nil {
		p.tx.Rollback()
		p.tx = nil
		return fmt.Errorf("failed to update profile %v: %w", p.ID, err)
	}

	err = p.tx.Commit()
	if err != nil {
		p.tx.Rollback()
		p.tx = nil
		return fmt.Errorf("failed to commit transaction for profile %v: %w", p.ID, err)
	}

	p.tx = nil
	p.modified = false
	return nil
}

func (p Profile) String() string {
	enc, _ := json.Marshal(p)
	var res strings.Builder
	res.WriteString("Profile: ")
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
