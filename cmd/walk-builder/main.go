package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maxdollinger/walk.io/internal/builder"
	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/oci"
)

func main() {
	// Create filesystem builder components
	layerFlattener := fs.NewLayerFlattener()
	blockDeviceBuilder := fs.NewExt4Builder()

	// Create orchestrator
	orchestrator := fs.NewFSBuilderOrchestrator(layerFlattener, blockDeviceBuilder)

	// Create builder
	bldr := builder.NewBuilder(orchestrator)

	imageSource, err := oci.NewRegistryProvider("oven/bun:latest")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	imageDir := os.Getenv("WALKIO_OUT_DIR")
	if len(imageDir) == 0 {
		imageDir = "/var/lib/walkio/app"
	}

	buildOpts := builder.BuildOptions{
		OutputDir: imageDir,
		WorkDir:   os.TempDir(),
	}
	result, err := bldr.Build(context.Background(), imageSource, buildOpts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(result)
}
