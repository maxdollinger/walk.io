package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maxdollinger/walk.io/internal/builder"
	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/lock"
	"github.com/maxdollinger/walk.io/pkg/oci"
)

func main() {
	bldr := builder.NewBuilder(fs.NewLayerFlattener(), fs.NewNoOpFilesystemPreparer(), fs.NewNoOpBlockDeviceBuilder(), lock.NewNoOpLocker())
	registry, err := oci.NewRegistryProvider("hello-world:latest")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	result, err := bldr.Build(context.Background(), registry, builder.BuildOptions{OutputDir: "~/walkio/"})
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println(result)
}
