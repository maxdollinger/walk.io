package builder_test

import (
	"context"
	"fmt"
	"log"

	"github.com/maxdollinger/walk.io/internal/builder"
	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/lock"
	"github.com/maxdollinger/walk.io/pkg/oci"
)

// ExampleNewBuilder demonstrates how to create and use the builder
func ExampleNewBuilder() {
	// Create dependencies (using no-op implementations for this example)
	flattener := fs.NewNoOpLayerFlattener()
	preparer := fs.NewNoOpFilesystemPreparer()
	blockDeviceBuilder := fs.NewNoOpBlockDeviceBuilder()
	locker := lock.NewNoOpLocker()

	// Create builder with dependency injection
	bldr := builder.NewBuilder(
		flattener,
		preparer,
		blockDeviceBuilder,
		locker,
	)

	// Define the image source
	source := oci.NewNoOpImageProvider()

	// Build the image
	ctx := context.Background()
	result, err := bldr.Build(ctx, source, builder.BuildOptions{
		OutputDir: "/tmp/walk-images",
	})
	if err != nil {
		log.Fatalf("build failed: %v", err)
	}

	// Use the result
	fmt.Printf("Block device created: %s\n", result.BlockDevicePath)
	fmt.Printf("Cached: %v\n", result.Cached)
	fmt.Printf("Build time: %v\n", result.BuildTime)
}
