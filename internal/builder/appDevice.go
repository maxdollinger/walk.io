// Package builder implements a Libaray to create a blockdevice with the app data from an OCI Container
package builder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/oci"
	"github.com/opencontainers/go-digest"
)

type AppFSopts struct {
	OutputDir string
}

type BuildResult struct {
	BlockDevicePath string           // full path to .ext4 file
	SourceDigest    digest.Digest    // digest of source image
	ImageConfig     *oci.ImageConfig // config from image
	BuildTime       time.Duration    // time taken to build
	Cached          bool             // true if existing block device was reused
}

func BuildAppDevice(ctx context.Context, imageSource oci.OciImageSource, deviceBuilder fs.BlockDeviceBuilder, opts *AppFSopts) (*BuildResult, error) {
	startTime := time.Now()
	buildTimeStamp := startTime.Unix()

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	image, err := imageSource.GetImage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to provide image: %w", err)
	}

	digestHex := image.Digest.Hex()

	// build is fresh invoked so set the wanted to this build
	wantedFile := path.Join(opts.OutputDir, digestHex+".wanted")
	err = fs.WriteFileAtomic(wantedFile, []byte(strconv.FormatInt(buildTimeStamp, 10)), 0o644)
	if err != nil {
		return nil, fmt.Errorf("error writing wanted file: %w", err)
	}

	tmpDevicePath := path.Join(opts.OutputDir, digestHex+"_tmp.ext4")
	appDevice, err := deviceBuilder.NewDevice(ctx, fs.BlockDeviceOptions{
		OutputFilePath: tmpDevicePath,
		SizeBytes:      image.Manifest.Size * 3,
		Label:          "APP_FS",
	})
	if err != nil {
		return nil, fmt.Errorf("appfs from image %s: %w", digestHex, err)
	}

	mountDir, err := appDevice.Mount()
	if err != nil {
		return nil, fmt.Errorf("appfs from image %s: %w", digestHex, err)
	}
	defer appDevice.Unmount()

	err = fs.UnpackImage(ctx, image.Layers, mountDir)
	if err != nil {
		return nil, fmt.Errorf("appfs from image %s: %w", digestHex, err)
	}

	err = fs.WriteContainerConfig(ctx, image.Config, mountDir)
	if err != nil {
		return nil, fmt.Errorf("appfs from image %s: %w", digestHex, err)
	}

	if !isNewstBuild(wantedFile, buildTimeStamp) {
		return nil, errors.New("newer build detected not publishing")
	}

	// atomic publish of newest build
	appDevice.Unmount()
	outputFilePath := path.Join(opts.OutputDir, digestHex+".ext4")
	err = os.Rename(tmpDevicePath, outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("appf from image %s: %w", digestHex, err)
	}

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
