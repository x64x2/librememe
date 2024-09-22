package models

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type conn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	DriverName() string
}

type Base struct {
	table string
}

func (b Base) TableName() string {
	return b.table
}

func (b Base) Save(ctx context.Context, conn conn, p any, opts *SaveOpts) (bool, error) {
	if opts == nil {
		opts = &SaveOpts{}
	}

	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = reflect.Indirect(v)
	}
	nf := t.NumField()
	cols := make([]string, nf)
	placeholders := make([]string, nf)
	vals := make([]any, nf)
	var idCol string
	n := 0
	for i := 0; i < nf; i++ {
		sf := t.Field(i)

		tag := sf.Tag.Get("db")
		if sf.Tag.Get("db_is_id") == "1" {
			idCol = tag
		}

		if !sf.IsExported() || tag == "" || tag == "-" {
			continue
		}

		cols[n] = tag
		switch conn.DriverName() {
		case "pgx":
			placeholders[n] = "$" + strconv.Itoa(n+1)
		case "sqlite":
			placeholders[n] = "?"
		}
		vals[n] = v.Field(i).Interface()
		n++
	}
	placeholders = placeholders[:n]
	cols = cols[:n]
	vals = vals[:n]

	builder := strings.Builder{}

	switch conn.DriverName() {
	case "pgx":
		builder.WriteString("INSERT INTO ")
	case "sqlite":
		switch {
		case opts.OnDuplicateIgnore:
			builder.WriteString("INSERT OR IGNORE INTO ")
		case opts.OnDuplicateReplace:
			builder.WriteString("INSERT OR REPLACE INTO ")
		default:
			builder.WriteString("INSERT INTO ")
		}
	}
	builder.WriteString(b.table)
	builder.WriteString(" (")
	for i, c := range cols {
		if i == 0 {
			builder.WriteString(c)
		} else {
			builder.WriteString(", " + c)
		}
	}
	builder.WriteString(") VALUES (")
	for i, p := range placeholders {
		if i == 0 {
			builder.WriteString(p)
		} else {
			builder.WriteString(", " + p)
		}
	}
	builder.WriteRune(')')

	switch conn.DriverName() {
	case "pgx":
		switch {
		case opts.OnDuplicateIgnore:
			builder.WriteString(" ON CONFLICT DO NOTHING")
		case opts.OnDuplicateReplace:
			builder.WriteString(" ON CONFLICT (" + idCol + ") DO UPDATE SET ")
			for i := range cols {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(cols[i] + " = " + placeholders[i])
			}
		}
	}

	res, err := conn.ExecContext(ctx, builder.String(), vals...)
	if err != nil {
		return false, fmt.Errorf("database error while saving: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to count affected rows: %w", err)
	}
	return count > 0, nil
}

type SaveOpts struct {
	OnDuplicateReplace bool
	OnDuplicateIgnore  bool
}

func (b Base) Update(ctx context.Context, conn conn, p any, updateCols []string) error {
	saved, err := b.Save(ctx, conn, p, &SaveOpts{
		OnDuplicateIgnore: true,
	})
	if err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}
	if saved {
		return nil
	}

	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = reflect.Indirect(v)
	}

	updates := make([]string, len(updateCols))
	vals := make([]any, len(updateCols)+1)
	var idCol string
	var idVal any
	n := 0
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		tag := sf.Tag.Get("db")
		if sf.Tag.Get("db_is_id") == "1" {
			idCol = tag
			idVal = v.Field(i).Interface()
		}

		if !slices.Contains(updateCols, tag) {
			continue
		}

		switch conn.DriverName() {
		case "pgx":
			updates[n] = tag + " = $" + strconv.Itoa(n+1)
		case "sqlite":
			updates[n] = tag + " = ?"
		}
		vals[n] = v.Field(i).Interface()
		n++
	}
	if n != len(updateCols) {
		panic("invalid updateCols")
	}
	vals[n] = idVal

	var sql string
	switch conn.DriverName() {
	case "pgx":
		sql = fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d", b.table, strings.Join(updates, ", "), idCol, n+1)
	case "sqlite":
		sql = fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", b.table, strings.Join(updates, ", "), idCol)
	}
	res, err := conn.ExecContext(ctx, sql, vals...)
	if err != nil {
		return fmt.Errorf("database error while updating: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count updated rows: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("no rows updated")
	}

	return nil
}

func IsValidID(val string) bool {
	if len(val) != 36 || val == db.NullUUID {
		return false
	}
	for i := 0; i < len(val); i++ {
		switch i {
		case 8, 13, 18, 23:
			if val[i] != '-' {
				return false
			}
		case 14:
			if val[i] != '4' {
				return false
			}
		case 19:
			switch val[i] {
			case '8', '9', 'A', 'a', 'B', 'b':
			default:
				return false
			}
		default:
			if !((val[i] >= 'a' && val[i] <= 'f') ||
				(val[i] >= 'A' && val[i] <= 'F') ||
				(val[i] >= '0' && val[i] <= '9')) {
				return false
			}
		}
	}
	return true
}
