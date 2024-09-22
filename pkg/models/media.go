package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

)

var MediaBase = Base{
	table: "media_v2",
}

type Media struct {
	ID               string `db:"media_id" db_is_id:"1"`
	Source           int    `db:"media_source"`
	SourceID         string `db:"media_source_id"`
	Type             string `db:"media_type"`
	Visible          uint8  `db:"media_visible"`
	Location         string `db:"media_location"`
	OriginalLocation string `db:"media_original_location"`
	Preview          string `db:"media_preview"`
	OriginalPreview  string `db:"media_original_preview"`
	Hash             []byte `db:"media_hash"`
	PreviewHash      []byte `db:"media_preview_hash"`

	modified bool
}

func NewMedia(source int, sourceID string) Media {
	id := uuid.Must(uuid.NewRandom())
	return Media{
		ID:       id.String(),
		Source:   source,
		SourceID: sourceID,
	}
}

func LoadMedia(ctx context.Context, mediaID string, tx *sqlx.Tx) (*Media, error) {
	if tx == nil {
		return queryMedia(ctx, db.Get(), `media_id = `+db.GetPlaceholder(), mediaID)
	}

	return queryMedia(ctx, tx, `media_id = `+db.GetPlaceholder(), mediaID)
}

func FindMedia(ctx context.Context, source int, sourceID string, tx *sqlx.Tx) (*Media, error) {
	if tx == nil {
		return findMedia(ctx, db.Get(), source, sourceID)
	}

	return findMedia(ctx, tx, source, sourceID)
}

func findMedia(ctx context.Context, conn conn, source int, sourceID string) (*Media, error) {
	placeholders := db.GetPlaceholders(2)
	return queryMedia(ctx, conn,
		`media_source = `+placeholders[0]+` AND media_source_id = `+placeholders[1],
		source, sourceID,
	)
}

func queryMedia(ctx context.Context, conn conn, where string, params ...any) (*Media, error) {
	q := `
SELECT
	media_id, media_source, media_source_id,
	media_type, media_visible,
	media_location, media_original_location,
	media_preview, media_original_preview,
	media_hash, media_preview_hash
FROM ` + MediaBase.TableName() + `
WHERE ` + where

	m := &Media{}

	row := conn.QueryRowContext(ctx, q, params...)
	err := row.Scan(
		&m.ID, &m.Source, &m.SourceID,
		&m.Type, &m.Visible,
		&m.Location, &m.OriginalLocation,
		&m.Preview, &m.OriginalPreview,
		&m.Hash, &m.PreviewHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading row: %w", err)
	}

	return m, nil
}

func (m *Media) SetModified() {
	m.modified = true
}

func (m Media) Valid() error {
	if !IsValidID(m.ID) {
		return errors.New("media ID is empty")
	}
	if !db.IsValidSource(m.Source) {
		return errors.New("media source is invalid")
	}
	if m.SourceID == "" {
		return errors.New("media source ID is empty")
	}
	if m.Type == "" {
		return errors.New("media type is empty")
	}
	switch strings.ToLower(m.Type) {
	case "photo", "video", "gif", "audio":
	default:
		return fmt.Errorf("media type '%s' is invalid", m.Type)
	}
	return nil
}

func (m *Media) Save(ctx context.Context, tx *sqlx.Tx) (err error) {
	if !m.modified {
		return nil
	}

	if tx == nil {
		_, err = MediaBase.Save(ctx, db.Get(), m, &SaveOpts{
			OnDuplicateReplace: true,
		})
	} else {
		_, err = MediaBase.Save(ctx, tx, m, &SaveOpts{
			OnDuplicateReplace: true,
		})
	}
	if err != nil {
		return err
	}

	m.modified = false
	return nil
}

func (m *Media) AddLocation(ctx context.Context, location string, originalLocation string, hash []byte) error {
	m.Location = location
	m.OriginalLocation = originalLocation

	if len(hash) == 0 {
		return m.doAddLocationPreview(ctx, []string{"media_location", "media_original_location"})
	}

	m.Hash = hash
	return m.doAddLocationPreview(ctx, []string{"media_location", "media_original_location", "media_hash"})
}

func (m *Media) AddPreview(ctx context.Context, preview string, originalPreview string, hash []byte) error {
	m.Preview = preview
	m.OriginalPreview = originalPreview

	if len(hash) == 0 {
		return m.doAddLocationPreview(ctx, []string{"media_preview", "media_original_preview"})
	}

	m.PreviewHash = hash
	return m.doAddLocationPreview(ctx, []string{"media_preview", "media_original_preview", "media_preview_hash"})
}

func (m *Media) doAddLocationPreview(ctx context.Context, updateCols []string) (err error) {
	tx, err := db.Get().BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for media %v: %w", m.ID, err)
	}

	err = MediaBase.Update(ctx, tx, m, updateCols)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update media %v: %w", m.ID, err)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction for media %v: %w", m.ID, err)
	}

	m.modified = false
	return nil
}

type MediaCol []string

func (mc MediaCol) Equal(other MediaCol) bool {
	if len(mc) != len(other) {
		return false
	}

	for i := 0; i < len(mc); i++ {
		if mc[i] != other[i] {
			return false
		}
	}

	return true
}

func (mc MediaCol) Value() (driver.Value, error) {
	if len(mc) == 0 {
		return nil, nil
	}

	b := strings.Builder{}

	driver := db.Get().DriverName()
	switch driver {
	case "pgx":
		b.WriteRune('{')
	case "sqlite":
		b.WriteString(`["`)
	default:
		return nil, errors.New("unsupported driver")
	}

	var wrote bool
	for _, v := range mc {
		if v == "" {
			continue
		}

		if wrote {
			switch driver {
			case "pgx":
				b.WriteRune(',')
			case "sqlite":
				b.WriteString(`","`)
			}
		}

		b.WriteString(v)
		wrote = true
	}

	switch driver {
	case "pgx":
		b.WriteRune('}')
	case "sqlite":
		b.WriteString(`"]`)
	}

	return b.String(), nil
}

func (mc *MediaCol) Scan(value any) error {
	var valStr string
	switch x := value.(type) {
	case nil:
		*mc = MediaCol{}
		return nil
	case string:
		valStr = x
	case []byte:
		valStr = string(x)
	default:
		return errors.New("invalid type for value")
	}

	if valStr == "" {
		*mc = MediaCol{}
		return nil
	}

	switch db.Get().DriverName() {
	case "pgx":
		if !strings.HasPrefix(valStr, "{") || !strings.HasSuffix(valStr, "}") {
			return errors.New("failed to parse postgres array")
		}
		valStr = valStr[1 : len(valStr)-1]
	case "sqlite":
		if !strings.HasPrefix(valStr, `["`) || !strings.HasSuffix(valStr, `"]`) {
			return errors.New("failed to parse JSON array")
		}
		valStr = valStr[2 : len(valStr)-2]
	default:
		return errors.New("unsupported driver")
	}

	if valStr == "" {
		*mc = MediaCol{}
		return nil
	}

	var ids []string
	switch db.Get().DriverName() {
	case "pgx":
		ids = strings.Split(valStr, ",")
	case "sqlite":
		ids = strings.Split(valStr, `","`)
	}

	res := make(MediaCol, len(ids))
	n := 0
	for i, s := range ids {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !IsValidID(s) {
			return fmt.Errorf("failed to parse ID %v in array: %s", i, s)
		}
		res[n] = s
		n++
	}
	res = res[:n]

	*mc = res
	return nil
}

func (mc MediaCol) String() string {
	enc, _ := json.Marshal(mc)
	return string(enc)
}

func (p Media) String() string {
	enc, _ := json.Marshal(p)
	var res strings.Builder
	res.WriteString("Media: ")
	if p.modified {
		res.WriteString("(modified) ")
	}
	res.Write(enc)
	return res.String()
}

func FindMediaByHash(ctx context.Context, hash []byte) (mediaID string, location string, err error) {
	conn := db.Get()

	placeholder := db.GetPlaceholder()
	var params []any
	switch conn.DriverName() {
	case "sqlite":
		params = []any{hash, hash}
	case "pgx":
		params = []any{hash}
	}

	q := `
SELECT media_id, media_location AS loc FROM ` + MediaBase.TableName() + ` WHERE media_hash = ` + placeholder + `
UNION
SELECT media_id, media_preview AS loc FROM ` + MediaBase.TableName() + ` WHERE media_preview_hash = ` + placeholder + `
`

	row := conn.QueryRowContext(ctx, q, params...)
	err = row.Scan(&mediaID, &location)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("error reading row: %w", err)
	}

	return mediaID, location, nil
}
