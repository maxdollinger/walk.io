package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/maxdollinger/walk.io/internal/builder"
	"github.com/maxdollinger/walk.io/internal/vm"
	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/oci"
)

const (
	WALKIO_BASE = "/var/lib/walkio/"
	APP_DIR     = WALKIO_BASE + "app"
	STATE_DIR   = WALKIO_BASE + "state"
	VM_DIR      = WALKIO_BASE + "vm"
)

func main() {
	appID, err := uuid.NewV7()
	if err != nil {
		fmt.Println("could not create apID: " + err.Error())
		os.Exit(1)
	}

	imageSource, err := oci.NewRegistryProvider("hello-world:latest")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	ctx := context.TODO()
	ext4Builder := fs.NewExt4Builder()
	result, err := builder.BuildAppDevice(ctx, imageSource, ext4Builder, &builder.AppFSopts{
		OutputDir: APP_DIR,
	})
	if err != nil {
		fmt.Printf("Building AppFS: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(result)

	stateDev, err := builder.BuildStateDevice(ctx, ext4Builder, &builder.StateFsOpts{
		AppID:     appID.String(),
		OutputDir: STATE_DIR,
		SizeBytes: 256 * 1024,
	})
	if err != nil {
		fmt.Printf("Building StateFS: %s\n", err)
		os.Exit(1)
	}

	vmConfig := vm.VMConfig{
		AppID:       appID.String(),
		AppFsPath:   result.BlockDevicePath,
		BaseVersion: "v0.1.0",
		VCPU:        1,
		Memory:      256,
		Timeout:     30 * time.Second,
	}

	vmRunner := vm.NewFirecrackerVM(VM_DIR)

	instance, err := vmRunner.Start(ctx, vmConfig, stateDev.Path())
	if err != nil {
		fmt.Printf("Failed to start VM: %s\n", err)
		os.Exit(1)
	}

	time.Sleep(5 * time.Second)

	status, err := vmRunner.Status(ctx, instance)
	if err != nil {
		fmt.Printf("Failed get VM Stats: %s\n", err)
		os.Exit(1)
	}

	fmt.Println(status)

	content, _ := os.ReadFile(instance.LogPath)
	fmt.Println(string(content))

	fmt.Println("finished")
}
