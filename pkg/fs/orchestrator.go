package fs

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/maxdollinger/walk.io/pkg/oci"
)

// FSBuilderOrchestrator coordinates the full filesystem build pipeline.
// It orchestrates layer flattening, config injection, and block device creation
// for any filesystem type (AppFs, StateFS, etc.).
type FSBuilderOrchestrator struct {
	layerFlattener     FsBuilder
	blockDeviceBuilder BlockDeviceBuilder
	logger             *slog.Logger
}

// NewFSBuilderOrchestrator creates a new filesystem builder orchestrator.
func NewFSBuilderOrchestrator(
	layerFlattener FsBuilder,
	blockDeviceBuilder BlockDeviceBuilder,
) *FSBuilderOrchestrator {
	return &FSBuilderOrchestrator{
		layerFlattener:     layerFlattener,
		blockDeviceBuilder: blockDeviceBuilder,
		logger:             slog.Default(),
	}
}

// BuildFromLayers builds a filesystem from OCI layers with custom config injection.
// This is the main method used by AppFs builder and image-based StateFS variants.
//
// Process:
//  1. Create temporary build directory
//  2. Extract and flatten OCI layers
//  3. Inject metadata using the provided BuilderConfig
//  4. Create ext4 block device
//  5. Atomically publish result (rename to final location)
func (o *FSBuilderOrchestrator) BuildFromLayers(
	ctx context.Context,
	layers []oci.Layer,
	config BuilderConfig,
	opts FSBuildOptions,
) (*FSBuildResult, error) {
	startTime := time.Now()

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	// Create temporary build directory
	buildRun := fmt.Sprintf("build-%d", startTime.Unix())
	buildDir := filepath.Join(opts.WorkDir, "walkio", "build", buildRun)
	o.logger.DebugContext(ctx, "creating build directory", "path", buildDir)

	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, fmt.Errorf("create build directory: %w", err)
	}
	defer func() {
		o.logger.DebugContext(ctx, "cleaning up build directory", "path", buildDir)
		if err := os.RemoveAll(buildDir); err != nil {
			o.logger.WarnContext(ctx, "failed to cleanup build directory", "error", err, "path", buildDir)
		}
	}()

	// Create rootfs directory
	rootfsDir := filepath.Join(buildDir, "rootfs")
	if err := os.MkdirAll(rootfsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create rootfs directory: %w", err)
	}

	// Step 1: Extract and flatten layers
	o.logger.InfoContext(ctx, "flattening layers", "count", len(layers))
	if err := o.layerFlattener.BuildFs(ctx, layers, rootfsDir); err != nil {
		return nil, fmt.Errorf("flatten layers: %w", err)
	}

	// Step 2: Inject configuration into rootfs
	o.logger.InfoContext(ctx, "injecting configuration into rootfs")
	if err := config.WriteConfig(ctx, rootfsDir); err != nil {
		return nil, fmt.Errorf("inject configuration: %w", err)
	}

	// Step 3: Create block device (using temporary name for atomic publish)
	tmpBuildFilePath := filepath.Join(opts.OutputDir, buildRun+".ext4")
	o.logger.InfoContext(ctx, "creating block device", "output", tmpBuildFilePath)

	blockDevice, err := o.blockDeviceBuilder.NewDevice(ctx, BlockDeviceOptions{
		SourceDirPath:  rootfsDir,
		BuildDirPath:   buildDir,
		OutputFilePath: tmpBuildFilePath,
		Label:          opts.Label,
	})
	if err != nil {
		return nil, fmt.Errorf("create block device: %w", err)
	}
	defer os.Remove(tmpBuildFilePath) // Clean up temp file if we don't publish

	// Step 4: Atomically publish result by renaming temp file
	// For now, we just return the path as-is (future: implement content-addressed naming)
	finalPath := blockDevice.Path

	duration := time.Since(startTime)
	o.logger.InfoContext(ctx, "filesystem built successfully",
		"path", finalPath,
		"size_mb", blockDevice.SizeBytes/1024/1024,
		"duration", duration)

	return &FSBuildResult{
		BlockDevicePath: finalPath,
		SizeBytes:       blockDevice.SizeBytes,
		BuildTime:       duration,
	}, nil
}

// BuildFromDirectory builds a filesystem from an existing directory with custom config injection.
// This is used by StateFS builder which creates an empty filesystem with metadata.
//
// Process:
//  1. Create temporary build directory
//  2. Copy sourceDir contents (or skip if empty for ephemeral filesystems)
//  3. Inject metadata using the provided BuilderConfig
//  4. Create ext4 block device
//  5. Return result
func (o *FSBuilderOrchestrator) BuildFromDirectory(
	ctx context.Context,
	sourceDir string,
	config BuilderConfig,
	opts FSBuildOptions,
) (*FSBuildResult, error) {
	startTime := time.Now()

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	// Create temporary build directory
	buildRun := fmt.Sprintf("build-%d", startTime.Unix())
	buildDir := filepath.Join(opts.WorkDir, "walkio", "build", buildRun)
	o.logger.DebugContext(ctx, "creating build directory", "path", buildDir)

	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, fmt.Errorf("create build directory: %w", err)
	}
	defer func() {
		o.logger.DebugContext(ctx, "cleaning up build directory", "path", buildDir)
		if err := os.RemoveAll(buildDir); err != nil {
			o.logger.WarnContext(ctx, "failed to cleanup build directory", "error", err, "path", buildDir)
		}
	}()

	// Create rootfs directory
	rootfsDir := filepath.Join(buildDir, "rootfs")
	if err := os.MkdirAll(rootfsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create rootfs directory: %w", err)
	}

	// Step 1: Copy source directory contents if provided
	if sourceDir != "" && sourceDir != "/" {
		o.logger.InfoContext(ctx, "copying source directory", "source", sourceDir)
		cmd := os.Getenv("COPY_CMD")
		if cmd == "" {
			cmd = "cp"
		}
		// Using os.CopyFS would be more idiomatic, but cp -a is more robust for this use case
		// For now, we'll implement a simple directory copy
		if err := copyDirectory(sourceDir, rootfsDir); err != nil {
			return nil, fmt.Errorf("copy source directory: %w", err)
		}
	}

	// Step 2: Inject configuration into rootfs
	o.logger.InfoContext(ctx, "injecting configuration into rootfs")
	if err := config.WriteConfig(ctx, rootfsDir); err != nil {
		return nil, fmt.Errorf("inject configuration: %w", err)
	}

	// Step 3: Create block device (using temporary name)
	tmpBuildFilePath := filepath.Join(opts.OutputDir, buildRun+".ext4")
	o.logger.InfoContext(ctx, "creating block device", "output", tmpBuildFilePath)

	blockDevice, err := o.blockDeviceBuilder.NewDevice(ctx, BlockDeviceOptions{
		SourceDirPath:  rootfsDir,
		BuildDirPath:   buildDir,
		OutputFilePath: tmpBuildFilePath,
		Label:          opts.Label,
	})
	if err != nil {
		return nil, fmt.Errorf("create block device: %w", err)
	}
	defer os.Remove(tmpBuildFilePath) // Clean up temp file if we don't publish

	// Step 4: Return result
	finalPath := blockDevice.Path

	duration := time.Since(startTime)
	o.logger.InfoContext(ctx, "filesystem built successfully",
		"path", finalPath,
		"size_mb", blockDevice.SizeBytes/1024/1024,
		"duration", duration)

	return &FSBuildResult{
		BlockDevicePath: finalPath,
		SizeBytes:       blockDevice.SizeBytes,
		BuildTime:       duration,
	}, nil
}

// copyDirectory recursively copies contents from src to dst.
// This is a simple implementation for build-time copying.
func copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}
