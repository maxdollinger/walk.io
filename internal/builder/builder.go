// Package builder implements a Libaray to create a blockdevice with the app data from an OCI Container
package builder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/oci"
	"github.com/opencontainers/go-digest"
)

type Builder interface {
	Build(ctx context.Context, provider oci.OciImageSource, opts BuildOptions) (*BuildResult, error)
}

type BuildOptions struct {
	OutputDir string // where to place final .ext4 files
	WorkDir   string
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
	logger             *slog.Logger
}

func NewBuilder(
	fsBuilder fs.FsBuilder,
	configInjector fs.ConfigWriter,
	blockDeviceBuilder fs.BlockDeviceBuilder,
) Builder {
	return &builder{
		fsBuilder:          fsBuilder,
		configWriter:       configInjector,
		blockDeviceBuilder: blockDeviceBuilder,
		logger:             slog.Default(),
	}
}

func (b *builder) Build(ctx context.Context, provider oci.OciImageSource, opts BuildOptions) (*BuildResult, error) {
	startTime := time.Now()
	buildTimeStamp := startTime.Unix()

	b.logger.InfoContext(ctx, "starting build", "providerInfo", provider.Info())

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	image, err := provider.GetImage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to provide image: %w", err)
	}

	digestHex := image.Digest.Hex()
	b.logger = b.logger.With("digest", digestHex)
	b.logger.InfoContext(ctx, "image fetched", "layers", len(image.Layers))

	// build is fresh invoked so set the wanted to this build
	wantedFile := path.Join(opts.OutputDir, digestHex+".wanted")
	err = fs.WriteFileAtomic(wantedFile, []byte(strconv.FormatInt(buildTimeStamp, 10)), 0o644)
	if err != nil {
		return nil, fmt.Errorf("error writing wanted file: %w", err)
	}

	buildRun := fmt.Sprintf("%s-%d", digestHex, buildTimeStamp)
	buildDir := filepath.Join(opts.WorkDir, "walkio", "build", buildRun)
	b.logger.DebugContext(ctx, "creating build directory", "path", buildDir)

	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create build directory: %w", err)
	}
	defer func() {
		b.logger.DebugContext(ctx, "cleaning up build directory", "path", buildDir)
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

	// to ensur atomicity of the later rename the tpm build output is placed in the the dir as the final file
	tmpBuildFilePath := filepath.Join(opts.OutputDir, buildRun+".ext4")
	b.logger.InfoContext(ctx, "creating block device", "output", tmpBuildFilePath)
	tmpBlockDevice, err := b.blockDeviceBuilder.NewDevice(ctx, fs.BlockDeviceOptions{
		SourceDirPath:  rootfsDir,
		BuildDirPath:   buildDir,
		OutputFilePath: tmpBuildFilePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create block device: %w", err)
	}
	defer os.Remove(tmpBuildFilePath)

	if !isNewstBuild(wantedFile, buildTimeStamp) {
		return nil, errors.New("newer build detected not publishing")
	}

	// atomic publish of newest build
	outputFilePath := path.Join(opts.OutputDir, digestHex+".ext4")
	err = os.Rename(tmpBlockDevice.Path, outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("error publishing buildresult: %w", err)
	}

	b.logger.InfoContext(ctx, "build completed successfully",
		"size_mb", tmpBlockDevice.SizeBytes/1024/1024,
		"duration", time.Since(startTime))

	return &BuildResult{
		BlockDevicePath: outputFilePath,
		SourceDigest:    image.Digest,
		ImageConfig:     image.Config,
		BuildTime:       time.Since(startTime),
		Cached:          false,
	}, nil
}

func isNewstBuild(filePath string, timestamp int64) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return true
	}

	ts, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return true
	}

	return ts <= timestamp
}
