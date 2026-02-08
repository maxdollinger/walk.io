package db

import (
	"context"
	"database/sql"
	"time"
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
	// TODO
	return nil, nil
}

func GetQueuedJobs(ctx context.Context, walkDB *sql.DB) ([]BuildJob, error) {
	// TODO
	return nil, nil
}
