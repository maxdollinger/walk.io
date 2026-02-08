package statefs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/maxdollinger/walk.io/pkg/fs"
)

// StateFSBuilder creates StateFS block devices.
type StateFSBuilder struct {
	fsOrchestrator *fs.FSBuilderOrchestrator
	logger         *slog.Logger
}

// NewStateFSBuilder creates a new StateFS builder.
func NewStateFSBuilder(fsOrchestrator *fs.FSBuilderOrchestrator) *StateFSBuilder {
	return &StateFSBuilder{
		fsOrchestrator: fsOrchestrator,
		logger:         slog.Default(),
	}
}

// BuildStateFS creates a new empty StateFS block device.
// The StateFS serves as a writable layer on top of the read-only AppFs.
func (b *StateFSBuilder) BuildStateFS(
	ctx context.Context,
	appID string,
	persistent bool,
	opts StateFSBuildOptions,
) (*StateFSInstance, error) {
	id := uuid.New().String()
	createdAt := time.Now()

	b.logger.InfoContext(ctx, "building statefs",
		"id", id,
		"app_id", appID,
		"size_mb", opts.SizeBytes/1024/1024,
		"persistent", persistent)

	// Create config for StateFS metadata injection
	config := &StateFSConfig{
		AppID:      appID,
		CreatedAt:  createdAt,
		Persistent: persistent,
	}

	// Build from empty directory
	fsResult, err := b.fsOrchestrator.BuildFromDirectory(
		ctx,
		"", // empty source directory
		config,
		fs.FSBuildOptions{
			OutputDir: opts.OutputDir,
			WorkDir:   opts.WorkDir,
			Label:     fmt.Sprintf("state-%s", id[:8]),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build statefs: %w", err)
	}

	b.logger.InfoContext(ctx, "statefs built successfully",
		"id", id,
		"path", fsResult.BlockDevicePath,
		"size_mb", fsResult.SizeBytes/1024/1024)

	return &StateFSInstance{
		ID:              id,
		AppID:           appID,
		BlockDevicePath: fsResult.BlockDevicePath,
		SizeBytes:       fsResult.SizeBytes,
		Persistent:      persistent,
		CreatedAt:       createdAt,
	}, nil
}
