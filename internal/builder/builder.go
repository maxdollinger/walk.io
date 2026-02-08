// Package builder implements a Libaray to create a blockdevice with the app data from an OCI Container
package builder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
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
	fsOrchestrator *fs.FSBuilderOrchestrator
	logger         *slog.Logger
}

func NewBuilder(fsOrchestrator *fs.FSBuilderOrchestrator) Builder {
	return &builder{
		fsOrchestrator: fsOrchestrator,
		logger:         slog.Default(),
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

	// Create app config writer for this image
	appConfigWriter := fs.NewAppConfigWriter(image.Config)

	// Use orchestrator to build filesystem from layers
	fsResult, err := b.fsOrchestrator.BuildFromLayers(
		ctx,
		image.Layers,
		appConfigWriter,
		fs.FSBuildOptions{
			OutputDir: opts.OutputDir,
			WorkDir:   opts.WorkDir,
			Label:     digestHex[:12],
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build filesystem: %w", err)
	}

	if !isNewstBuild(wantedFile, buildTimeStamp) {
		return nil, errors.New("newer build detected not publishing")
	}

	// atomic publish of newest build
	outputFilePath := path.Join(opts.OutputDir, digestHex+".ext4")
	err = os.Rename(fsResult.BlockDevicePath, outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("error publishing buildresult: %w", err)
	}

	b.logger.InfoContext(ctx, "build completed successfully",
		"size_mb", fsResult.SizeBytes/1024/1024,
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
