package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrate runs all pending goose migrations from the given directory.
func Migrate(ctx context.Context, connStr string, dir string) error {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("migrate: failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrate: failed to set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, dir); err != nil {
		return fmt.Errorf("migrate: failed to run migrations: %w", err)
	}

	return nil
}
