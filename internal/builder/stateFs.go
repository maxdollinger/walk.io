package builder

import (
	"context"
	"fmt"
	"path"

	"github.com/google/uuid"
	"github.com/maxdollinger/walk.io/pkg/fs"
)

type StateFsOpts struct {
	AppID     string
	SizeBytes int64
	OutputDir string
}

func BuildStateFs(ctx context.Context, blockDeviceBuilder fs.BlockDeviceBuilder, opts *StateFsOpts) (string, error) {
	uuid, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("building statefs for %s: %w", opts.AppID, err)
	}

	devicePath := path.Join(opts.OutputDir, opts.AppID+"_"+uuid.String())
	blockDevice, err := blockDeviceBuilder.NewDevice(ctx, fs.BlockDeviceOptions{
		SizeBytes:      opts.SizeBytes,
		OutputFilePath: devicePath,
	})
	if err != nil {
		return "", fmt.Errorf("building statefs for %s: %w", opts.AppID, err)
	}

	return blockDevice.Path(), nil
}
