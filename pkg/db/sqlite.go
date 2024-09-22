package db

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
	"modernc.org/sqlite"
)

func registerSqliteFuncs() {
	sqlite.MustRegisterScalarFunction("gen_random_uuid", 0, func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		u, err := uuid.NewRandom()
		if err != nil {
			return nil, fmt.Errorf("failed to generate UUID: %w", err)
		}

		return u.String(), nil
	})
}
