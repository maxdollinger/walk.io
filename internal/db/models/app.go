package db

import (
	"context"
	"database/sql"
	"time"
)

type App struct {
	ID               string // unique application identifier
	ImageName        string // OCI image name (e.g., "nginx:latest")
	EnvJson          string // JSON-encoded environment variables
	ArgsJson         string // JSON-encoded command arguments
	WorkDir          string // working directory in container
	KernelPath       string // path to firecracker kernel for this app
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
