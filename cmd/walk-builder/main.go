package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/maxdollinger/walk.io/internal/builder"
	"github.com/maxdollinger/walk.io/internal/vm"
	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/oci"
	"github.com/maxdollinger/walk.io/pkg/utils"
)

const (
	WALKIO_BASE = "/var/lib/walkio/"
	APP_DIR     = WALKIO_BASE + "app"
	STATE_DIR   = WALKIO_BASE + "state"
	VM_DIR      = WALKIO_BASE + "vm"
)

func main() {
	startTime := time.Now()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.TODO()

	appID, err := uuid.NewV7()
	if err != nil {
		fmt.Println("could not create apID: " + err.Error())
		os.Exit(1)
	}
	logger = logger.With("appID", appID.String())

	imageSource, err := oci.NewRegistryProvider("hello-world:latest")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	logger = logger.With("imageSource", imageSource.Info())

	ext4Builder := fs.NewExt4Builder()
	appResult, err := builder.BuildAppDevice(ctx, imageSource, ext4Builder, &builder.AppFSopts{
		OutputDir: APP_DIR,
	})
	if err != nil {
		fmt.Printf("Building AppFS: %s\n", err)
		os.Exit(1)
	}
	logger = logger.With("appDevice", appResult)

	stateResult, err := builder.BuildStateDevice(ctx, ext4Builder, &builder.StateFsOpts{
		AppID:     appID.String(),
		OutputDir: STATE_DIR,
		SizeBytes: 0,
	})
	if err != nil {
		fmt.Printf("Building StateFS: %s\n", err)
		os.Exit(1)
	}
	logger = logger.With("stateDevice", stateResult)

	vmConfig := vm.VMConfig{
		AppID:       appID.String(),
		AppFsPath:   appResult.BlockDevicePath,
		BaseVersion: "v0.1.1",
		VCPU:        2,
		Memory:      256,
		Timeout:     30 * time.Second,
	}

	vmRunner := vm.NewFirecrackerVM(VM_DIR)

	instance, err := vmRunner.Start(ctx, vmConfig, stateResult.BlockDevicePath)
	if err != nil {
		fmt.Printf("Failed to start VM: %s\n", err)
		os.Exit(1)
	}

	logger = logger.With(
		"vm_id", instance.ID,
		"vm_pid", instance.PID,
		"vm_statedAt", instance.StartedAt,
	)

	status := vm.VMStatusRunning
	for status == vm.VMStatusRunning {
		status, err = vmRunner.Status(ctx, instance)
		if err != nil {
			fmt.Printf("Failed get VM Stats: %s\n", err)
			os.Exit(1)
		}
		time.Sleep(50 * time.Millisecond)
	}

	logger.Info("Finished execution", "vm_status", status, "exec_time", time.Since(startTime).Seconds())

	fmt.Println("---- VM-Logs -----")
	fmt.Println("")
	_ = utils.TailPollUntilIdle(instance.LogPath, os.Stdout, 500*time.Millisecond, 20*time.Millisecond)
}
