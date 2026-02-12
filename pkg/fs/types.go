package fs

import (
	"context"
)

type BlockDeviceBuilder interface {
	// Creates an ext4 image with minimal size for the content
	NewDevice(ctx context.Context, opts BlockDeviceOptions) (BlockDevice, error)
}

type BlockDeviceOptions struct {
	OutputFilePath string // Path of the dir the device is created in
	SizeBytes      int64  // Blockdevice size in bytes (for journaled block devices greater than 6144 bytes)
	Label          string // filesystem label (optional)
}

type BlockDevice interface {
	Mount() (string, error)
	Unmount() error
	SizeBytes() int64
	Label() string
	Path() string
}
