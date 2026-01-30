package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type BuildJob struct {
	ID              string     `json:"id"`
	AppID           string     `json:"app_id"`
	ImageName       string     `json:"image_name"`
	Status          string     `json:"status"`
	Digest          *string    `json:"digest,omitempty"`
	BlockDevicePath *string    `json:"block_device_path,omitempty"`
	Error           *string    `json:"error,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func InsertBuildJob(ctx context.Context, walkDB *sql.DB, appID, imageName string) (*BuildJob, error) {
	jobID, err := uuid.NewV7() // You'll need to implement this UUID generator
	if err != nil {
		return nil, fmt.Errorf("error generating buildjob uuid: %w", err)
	}
	now := time.Now().Unix()

	query := `
		INSERT INTO build_jobs (id, app_id, image_name, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = walkDB.ExecContext(ctx, query, jobID, appID, imageName, "queued", now)
	if err != nil {
		return nil, err
	}

	createdAtTime := time.Unix(now, 0)
	return &BuildJob{
		ID:        jobID.String(),
		AppID:     appID,
		ImageName: imageName,
		Status:    "queued",
		CreatedAt: createdAtTime,
	}, nil
}
