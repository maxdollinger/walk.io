package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/maxdollinger/walk.io/pkg/utils"
)

type firecracker struct {
	vmsDir string // directory for control
	logger *slog.Logger
}

func NewFirecrackerVM(vmsDir string) VMRuntime {
	return &firecracker{
		vmsDir: vmsDir,
		logger: slog.Default(),
	}
}

func (f *firecracker) Start(ctx context.Context, config VMConfig, stateDevPath string) (*VMInstance, error) {
	id, err := utils.NewUUID7()
	if err != nil {
		return nil, fmt.Errorf("generate vm id: %w", err)
	}

	f.logger.InfoContext(ctx, "starting firecracker vm",
		"id", id,
		"vcpu", config.VCPU,
		"memory_mb", config.Memory)

	if err := f.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid vm config: %w", err)
	}

	vmDir := filepath.Join(f.vmsDir, id)
	if err := os.MkdirAll(vmDir, 0o700); err != nil {
		return nil, fmt.Errorf("create socket directory: %w", err)
	}

	socketPath := filepath.Join(vmDir, "api.sock")
	configPath := filepath.Join(vmDir, "config.json")
	fcConfig := f.buildFirecrackerConfig(config, stateDevPath)
	if err := f.writeFirecrackerConfig(configPath, fcConfig); err != nil {
		f.cleanup(vmDir)
		return nil, fmt.Errorf("write firecracker config: %w", err)
	}

	logPath := filepath.Join(vmDir, "firecracker.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		f.cleanup(vmDir)
		return nil, fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()

	cmd := exec.CommandContext(ctx, config.GetFirecrackerPath(), "--api-sock", socketPath, "--config-file", configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		f.cleanup(vmDir)
		return nil, fmt.Errorf("start firecracker process: %w", err)
	}

	pid := cmd.Process.Pid
	f.logger.InfoContext(ctx, "firecracker process started",
		"id", id,
		"pid", pid)

	socketSpawnTimeout := time.Second
	if err := f.waitForSocket(ctx, socketPath, socketSpawnTimeout); err != nil {
		f.cleanup(vmDir)
		return nil, fmt.Errorf("firecracker healthcheck failed: %w", err)
	}

	f.logger.InfoContext(ctx, "firecracker vm started successfully",
		"id", id,
		"pid", pid,
		"socket", socketPath)

	return &VMInstance{
		ID:           id,
		PID:          pid,
		SocketPath:   socketPath,
		ConfigPath:   configPath,
		LogPath:      logPath,
		StateDevPath: stateDevPath,
		VMConfig:     &config,
		Meta:         make(map[string]any),
		StartedAt:    time.Now(),
	}, nil
}

func (f *firecracker) Stop(ctx context.Context, instance *VMInstance) error {
	f.logger.InfoContext(ctx, "stopping firecracker vm", "id", instance.ID)

	// TODO: Send shutdown command via Firecracker API socket
	// For now, we just clean up the socket directory

	// Clean up socket directory
	vmSockDir := filepath.Dir(instance.SocketPath)
	if err := os.RemoveAll(vmSockDir); err != nil {
		f.logger.WarnContext(ctx, "failed to cleanup socket directory", "error", err)
		return err
	}

	f.logger.InfoContext(ctx, "firecracker vm stopped", "id", instance.ID)
	return nil
}

func (f *firecracker) Status(ctx context.Context, instance *VMInstance) (VMStatus, error) {
	// TODO change this to check for socketFile and send a healthcheck query
	if instance.PID <= 0 {
		return VMStatusStopped, nil
	}

	proc, err := os.FindProcess(instance.PID)
	if err != nil {
		return VMStatusStopped, nil
	}
	defer proc.Release()

	// Try to send signal 0 to check if process is alive
	if err := proc.Signal(os.Signal(nil)); err != nil {
		return VMStatusStopped, nil
	}

	return VMStatusRunning, nil
}

func (f *firecracker) validateConfig(config *VMConfig) error {
	if _, err := os.Stat(config.GetRootFSPath()); err != nil {
		return fmt.Errorf("rootfs not found at %s: %w", config.GetRootFSPath(), err)
	}
	if _, err := os.Stat(config.AppFsPath); err != nil {
		return fmt.Errorf("appfs not found at %s: %w", config.AppFsPath, err)
	}
	if _, err := os.Stat(config.GetKernelPath()); err != nil {
		return fmt.Errorf("kernel not found at %s: %w", config.GetKernelPath(), err)
	}
	if config.VCPU <= 0 {
		config.VCPU = 1
	}
	if config.Memory <= 0 {
		config.Memory = 128
	}
	return nil
}

func (f *firecracker) buildFirecrackerConfig(config VMConfig, stateDevPath string) map[string]any {
	return map[string]any{
		"boot-source": map[string]any{
			"kernel_image_path": config.GetKernelPath(),
			"boot_args":         "console=ttyS0 reboot=k panic=1 init=/vmax/init",
		},
		"machine-config": map[string]any{
			"vcpu_count":   config.VCPU,
			"mem_size_mib": config.Memory,
			"smt":          false,
		},
		"drives": []map[string]any{
			// Drive 1: RootFS - system initialization (root device, read-only, shared)
			{
				"drive_id":       "rootfs",
				"path_on_host":   config.GetRootFSPath(),
				"is_root_device": true,
				"is_read_only":   true,
			},
			// Drive 2: AppFS - application code/data (secondary, read-only)
			{
				"drive_id":       "app",
				"path_on_host":   config.AppFsPath,
				"is_root_device": false,
				"is_read_only":   true,
			},
			// Drive 3: StateFS - runtime state (secondary, writable)
			{
				"drive_id":       "state",
				"path_on_host":   stateDevPath,
				"is_root_device": false,
				"is_read_only":   false,
			},
		},
	}
}

func (f *firecracker) writeFirecrackerConfig(path string, config map[string]any) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func (f *firecracker) waitForSocket(ctx context.Context, socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if _, err := os.Stat(socketPath); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("socket did not appear within %v", timeout)
			}
		}
	}
}

func (f *firecracker) cleanup(vmSockDir string) {
	// err := os.RemoveAll(vmSockDir)
	// if err != nil {
	// 	f.logger.Error("failed to cleanup vmSocketDir %s: %s", vmSockDir, err)
	// }
}
