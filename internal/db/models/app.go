package db

import (
	"context"
	"database/sql"
	"time"
)

type App struct {
	ID               string // unique application identifier
	Digest           string // OCI image digest (e.g., "sha256:abc123...")
	BaseVersion      string // base bundle version (e.g., "v1.0", "v2.0") references /var/lib/walkio/base/[version]
	StateFsSizeBytes int64  // size of StateFS in bytes (default 1GB)
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func UpsertApp(ctx context.Context, walkDB *sql.DB, app *App) error {
	// TODO
	return nil
}

func GetAppByID(ctx context.Context, walkDB *sql.DB, appID string) (*App, error) {
	// TODO
	return nil, nil
}
