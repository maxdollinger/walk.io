package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type App struct {
	ID        string            `json:"id"`
	ImageName string            `json:"image_name"`
	Env       map[string]string `json:"env,omitempty"`
	Args      []string          `json:"args,omitempty"`
	WorkDir   string            `json:"work_dir,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func UpsertApp(ctx context.Context, walkDB *sql.DB, app *App) error {
	envJSON, _ := json.Marshal(app.Env)
	argsJSON, _ := json.Marshal(app.Args)
	now := time.Now().Unix()
	query := `
		INSERT INTO apps (id, image_name, env_json, args_json, work_dir, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			image_name=excluded.image_name,
			env_json=excluded.env_json,
			args_json=excluded.args_json,
			work_dir=excluded.work_dir,
			updated_at=excluded.updated_at
	`
	_, err := walkDB.ExecContext(ctx, query,
		app.ID,
		app.ImageName,
		string(envJSON),
		string(argsJSON),
		app.WorkDir,
		now,
		now,
	)
	return err
}

func GetAppByID(ctx context.Context, walkDB *sql.DB, appID string) (*App, error) {
	var app App
	var envJSON, argsJSON string
	var createdAt, updatedAt int64
	query := `SELECT id, image_name, env_json, args_json, work_dir, created_at, updated_at FROM apps WHERE id = ?`
	err := walkDB.QueryRowContext(ctx, query, appID).Scan(
		&app.ID, &app.ImageName, &envJSON, &argsJSON, &app.WorkDir, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal([]byte(envJSON), &app.Env); err != nil {
		return nil, err
	}
	if err = json.Unmarshal([]byte(argsJSON), &app.Args); err != nil {
		return nil, err
	}

	app.CreatedAt = time.Unix(createdAt, 0)
	app.UpdatedAt = time.Unix(updatedAt, 0)

	return &app, nil
}
