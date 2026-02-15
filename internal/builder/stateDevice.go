package builder

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/maxdollinger/walk.io/pkg/fs"
)

type StateFsOpts struct {
	AppID     string
	SizeBytes int64
	OutputDir string
}

func BuildStateDevice(ctx context.Context, blockDeviceBuilder fs.BlockDeviceBuilder, opts *StateFsOpts) (*BuildResult, error) {
	startTime := time.Now()

	uuid, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("building statefs for %s: %w", opts.AppID, err)
	}

	devicePath := path.Join(opts.OutputDir, opts.AppID+"_"+uuid.String())
	_, err = blockDeviceBuilder.NewDevice(ctx, fs.BlockDeviceOptions{
		SizeBytes:      opts.SizeBytes,
		OutputFilePath: devicePath,
	})
	if err != nil {
		return nil, fmt.Errorf("building statefs for %s: %w", opts.AppID, err)
	}

	return &BuildResult{
		BlockDevicePath: devicePath,
		BuildTime:       time.Since(startTime),
		Cached:          false,
	}, nil
}
