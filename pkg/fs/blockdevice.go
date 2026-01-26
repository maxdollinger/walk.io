package fs

import (
	"context"
)

type BlockDeviceBuilder interface {
	// Creates an ext4 image with minimal size for the content
	NewDevice(ctx context.Context, opts BlockDeviceOptions) (*BlockDevice, error)
}

// BlockDeviceOptions specifies how to create a block device
type BlockDeviceOptions struct {
	SourceDir  string // prepared rootfs directory
	OutputPath string // where to write the .ext4 file
	Label      string // filesystem label (optional)
}

type BlockDevice struct {
	Path      string
	SizeBytes int64
	Label     string
}

type NoOpBlockDeviceBuilder struct{}

func NewNoOpBlockDeviceBuilder() *NoOpBlockDeviceBuilder {
	return &NoOpBlockDeviceBuilder{}
}

func (b *NoOpBlockDeviceBuilder) NewDevice(ctx context.Context, opts BlockDeviceOptions) (*BlockDevice, error) {
	// No-op: in real implementation, would create ext4 image
	return &BlockDevice{
		Path:      opts.OutputPath,
		SizeBytes: 1024 * 1024 * 100, // 100MB dummy size
		Label:     opts.Label,
	}, nil
}
