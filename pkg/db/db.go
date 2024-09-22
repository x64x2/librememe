package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"codeberg.org/biggestfan24/myfans/pkg/config"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

var (
	conn       *sqlx.DB
	migrations [6]func(ctx context.Context, db *sqlx.DB) error
	lock       sync.Mutex
)

func Connect(readOnly bool) (err error) {
	conf := config.Global.GetString(config.KeyDatabase)

	if conf == "" {
		dp, err := config.DataPath()
		if err != nil {
			return fmt.Errorf("data path '%s' is invalid", config.Global.GetString(config.KeyDataPath))
		}
		conf = "sqlite:" + filepath.Join(dp, "data.db")
	}

	switch {
	case strings.HasPrefix(conf, "postgres:"):
		conn, err = sqlx.Open("pgx", conf)
	case strings.HasPrefix(conf, "sqlite://"):
		conn, err = connectSqlite(conf, readOnly)
	case strings.HasPrefix(conf, "sqlite:"):
		conn, err = connectSqlite("sqlite://"+conf[7:], readOnly)
	default:
		conn, err = connectSqlite("sqlite://"+conf, readOnly)
	}

	return err
}

func connectSqlite(dsn string, readOnly bool) (*sqlx.DB, error) {
	registerSqliteFuncs()

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sqlite URL")
	}

	qs := url.Values{}
	if readOnly {
		qs.Set("mode", "ro")
	}
	qs.Set("_locking_mode", "EXCLUSIVE")
	qs.Set("_txlock", "immediate")
	qs.Add("_pragma", "busy_timeout(10000)")
	if readOnly {
		qs.Add("_pragma", "journal_mode(WAL)")
	} else {
		qs.Add("_pragma", "journal_mode(DELETE)")
	}

	dir := filepath.Dir(u.Path)
	err = os.MkdirAll(dir, 0o775)
	if err != nil {
		return nil, fmt.Errorf("failed to create path '%s' for the database: %w", dir, err)
	}
	return sqlx.Open("sqlite", u.Path+"?"+qs.Encode())
}

func Get() *sqlx.DB {
	return conn
}

func Migrate(ctx context.Context) (err error) {
	resTable := &struct {
		Name string
	}{}

	switch conn.DriverName() {
	case "pgx":
		err = conn.GetContext(ctx, resTable, `SELECT table_name AS name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'migrations'`)
	case "sqlite":
		err = conn.GetContext(ctx, resTable, `SELECT name FROM sqlite_master WHERE type = "table" AND name = "migrations"`)
	}
	if errors.Is(err, sql.ErrNoRows) {
		_, err := conn.ExecContext(ctx, `
CREATE TABLE migrations (version integer not null);
INSERT INTO migrations (version) VALUES (0);
`)
		if err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query for migrations table: %w", err)
	}

	resVersion := &struct {
		Version int
	}{}
	err = conn.GetContext(ctx, resVersion, `SELECT version FROM migrations`)
	if err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	for i := resVersion.Version; i < len(migrations); i++ {
		log.Infof("💾 Applying DB migration %d", i+1)

		err = migrations[i](ctx, conn)
		if err != nil {
			return fmt.Errorf("failed to perform migration %d: %w", i+1, err)
		}

		switch conn.DriverName() {
		case "pgx":
			_, err = conn.Exec("UPDATE migrations SET version = $1", i+1)
		case "sqlite":
			_, err = conn.Exec("UPDATE migrations SET version = ?", i+1)
		}
		if err != nil {
			return fmt.Errorf("failed to update migrations table for version %d: %w", i+1, err)
		}
	}

	return nil
}

func GetPlaceholder() string {
	switch conn.DriverName() {
	case "sqlite":
		return "?"
	case "pgx":
		return "$1"
	default:
		panic("invalid driver")
	}
}

func GetPlaceholders(n int) []string {
	placeholders := make([]string, n)
	switch conn.DriverName() {
	case "sqlite":
		for i := 0; i < n; i++ {
			placeholders[i] = "?"
		}
	case "pgx":
		for i := 0; i < n; i++ {
			placeholders[i] = "$" + strconv.Itoa(i+1)
		}
	}
	return placeholders
}
