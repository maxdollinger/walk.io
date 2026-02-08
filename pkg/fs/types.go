package fs

import (
	"context"
	"time"
)

// BuilderConfig abstracts what metadata/config to inject into a filesystem.
// Different filesystem types (AppFs, StateFS) implement this interface to inject
// their own metadata/configuration into the rootfs before block device creation.
type BuilderConfig interface {
	// WriteConfig injects custom metadata into the rootfs directory.
	// This is called after layer flattening but before block device creation.
	WriteConfig(ctx context.Context, rootfsDir string) error
}

// FSBuildOptions specifies parameters for building a filesystem from layers or directory.
type FSBuildOptions struct {
	OutputDir string // directory to place final .ext4 file
	WorkDir   string // temporary directory for build artifacts
	Label     string // ext4 filesystem label (optional)
}

// FSBuildResult contains the output of a filesystem build operation.
type FSBuildResult struct {
	BlockDevicePath string        // path to generated .ext4 file
	SizeBytes       int64         // filesystem size in bytes
	BuildTime       time.Duration // time taken to build
}
