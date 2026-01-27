// Package builder implements a Libaray to create a blockdevice with the app data from an OCI Container
package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/lock"
	"github.com/maxdollinger/walk.io/pkg/oci"
	"github.com/opencontainers/go-digest"
)

type Builder interface {
	Build(ctx context.Context, provider oci.ImageProvider, opts BuildOptions) (*BuildResult, error)
}

type BuildOptions struct {
	OutputDir string // where to place final .ext4 files
}

// BuildResult contains information about the built artifact
type BuildResult struct {
	BlockDevicePath string           // full path to .ext4 file
	SourceDigest    digest.Digest    // digest of source image
	ImageConfig     *oci.ImageConfig // config from image
	BuildTime       time.Duration    // time taken to build
	Cached          bool             // true if existing block device was reused
}

type builder struct {
	fsBuilder          fs.FsBuilder
	configWriter       fs.ConfigWriter
	blockDeviceBuilder fs.BlockDeviceBuilder
	locker             lock.Locker
	logger             *slog.Logger
}

func NewBuilder(
	fsBuilder fs.FsBuilder,
	configInjector fs.ConfigWriter,
	blockDeviceBuilder fs.BlockDeviceBuilder,
	locker lock.Locker,
) Builder {
	return &builder{
		fsBuilder:          fsBuilder,
		configWriter:       configInjector,
		blockDeviceBuilder: blockDeviceBuilder,
		locker:             locker,
		logger:             slog.Default(),
	}
}

func (b *builder) Build(ctx context.Context, provider oci.ImageProvider, opts BuildOptions) (*BuildResult, error) {
	startTime := time.Now()

	b.logger.InfoContext(ctx, "starting build", "providerInfo", provider.Info())

	image, err := provider.GetImage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to provide image: %w", err)
	}

	digestHex := image.Digest.Hex()
	b.logger = b.logger.With("digest", digestHex)
	b.logger.InfoContext(ctx, "image fetched", "layers", len(image.Layers))

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.ext4", digestHex))

	if fileExists(outputPath) {
		b.logger.InfoContext(ctx, "using cached block device", "path", outputPath)
		return &BuildResult{
			BlockDevicePath: outputPath,
			SourceDigest:    image.Digest,
			ImageConfig:     image.Config,
			BuildTime:       time.Since(startTime),
			Cached:          true,
		}, nil
	}

	b.logger.DebugContext(ctx, "acquiring lock")
	lockHandle, err := b.locker.AcquireLock(ctx, image.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		if err := lockHandle.Release(); err != nil {
			b.logger.WarnContext(ctx, "failed to release lock", "error", err)
		}
	}()

	b.logger.DebugContext(ctx, "lock acquired")

	if fileExists(outputPath) {
		b.logger.InfoContext(ctx, "block device created while waiting for lock", "path", outputPath)
		return &BuildResult{
			BlockDevicePath: outputPath,
			SourceDigest:    image.Digest,
			ImageConfig:     image.Config,
			BuildTime:       time.Since(startTime),
			Cached:          true,
		}, nil
	}

	workDir := os.TempDir()
	buildDir := filepath.Join(workDir, "walkio", "build", digestHex)
	b.logger.DebugContext(ctx, "creating work directory", "path", buildDir)

	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	defer func() {
		b.logger.DebugContext(ctx, "cleaning up work directory", "path", buildDir)
		if err := os.RemoveAll(buildDir); err != nil {
			b.logger.WarnContext(ctx, "failed to cleanup work directory", "error", err, "path", buildDir)
		}
	}()

	rootfsDir := filepath.Join(buildDir, "rootfs")
	if err := os.MkdirAll(rootfsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	b.logger.InfoContext(ctx, "building app fs", "count", len(image.Layers))
	if err := b.fsBuilder.BuildFs(ctx, image.Layers, rootfsDir); err != nil {
		return nil, fmt.Errorf("failed to flatten layers: %w", err)
	}

	b.logger.InfoContext(ctx, "preparing rootfs with walk.io metadata")
	if err := b.configWriter.WriteConfig(ctx, rootfsDir, image.Config); err != nil {
		return nil, fmt.Errorf("failed to prepare rootfs: %w", err)
	}

	b.logger.InfoContext(ctx, "creating block device", "output", outputPath)
	blockDevice, err := b.blockDeviceBuilder.NewDevice(ctx, fs.BlockDeviceOptions{
		SourceDirPath:  rootfsDir,
		OutputFilePath: outputPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create block device: %w", err)
	}

	b.logger.InfoContext(ctx, "build completed successfully",
		"size_mb", blockDevice.SizeBytes/1024/1024,
		"duration", time.Since(startTime))

	return &BuildResult{
		BlockDevicePath: blockDevice.Path,
		SourceDigest:    image.Digest,
		ImageConfig:     image.Config,
		BuildTime:       time.Since(startTime),
		Cached:          false,
	}, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
