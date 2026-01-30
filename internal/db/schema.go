package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
)

//go:embed migration/*.sql
var migrationFiles embed.FS

func InitSchema(ctx context.Context, db *sql.DB) error {
	schema, err := migrationFiles.ReadFile("migration/001_initial.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = db.ExecContext(ctx, string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}
