package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// FirecrackerVM manages Firecracker microVM lifecycle.
// It handles VM creation, startup, shutdown, and status checks.
type FirecrackerVM interface {
	// Start launches a Firecracker microVM with the given configuration.
	// Returns a VMInstance on success, or an error on failure.
	Start(ctx context.Context, config VMConfig) (*VMInstance, error)

	// Stop terminates a running Firecracker VM.
	Stop(ctx context.Context, instance *VMInstance) error

	// Status checks the current state of a VM.
	Status(ctx context.Context, instance *VMInstance) (VMStatus, error)

	// Future: Pause, Resume, GetMetrics
}

type firecracker struct {
	binaryPath string // path to firecracker binary
	socketsDir string // directory for control sockets
	logger     *slog.Logger
}

// NewFirecrackerVM creates a new Firecracker VM manager.
// It reads the firecracker binary path from WALKIO_FIRECRACKER_BIN environment variable,
// or defaults to /usr/bin/firecracker if not set.
func NewFirecrackerVM(socketsDir string) FirecrackerVM {
	binaryPath := os.Getenv("WALKIO_FIRECRACKER_BIN")
	if binaryPath == "" {
		binaryPath = "/usr/bin/firecracker"
	}

	return &firecracker{
		binaryPath: binaryPath,
		socketsDir: socketsDir,
		logger:     slog.Default(),
	}
}

// Start launches a Firecracker microVM with the given configuration.
// Process:
//  1. Validate configuration (all paths exist)
//  2. Create socket directory for VM control
//  3. Generate Firecracker configuration JSON
//  4. Start Firecracker process
//  5. Wait for control socket to appear (healthcheck)
//  6. Return VMInstance on success
func (f *firecracker) Start(ctx context.Context, config VMConfig) (*VMInstance, error) {
	// Generate unique VM ID
	id, err := generateVMID()
	if err != nil {
		return nil, fmt.Errorf("generate vm id: %w", err)
	}

	f.logger.InfoContext(ctx, "starting firecracker vm",
		"id", id,
		"vcpu", config.VCPU,
		"memory_mb", config.Memory)

	// Step 1: Validate inputs
	if err := f.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid vm config: %w", err)
	}

	// Step 2: Create socket directory
	vmSockDir := filepath.Join(f.socketsDir, id)
	if err := os.MkdirAll(vmSockDir, 0o700); err != nil {
		return nil, fmt.Errorf("create socket directory: %w", err)
	}

	socketPath := filepath.Join(vmSockDir, "api.sock")

	// Step 3: Generate Firecracker configuration
	fcConfig := f.buildFirecrackerConfig(config, socketPath)
	configPath := filepath.Join(vmSockDir, "config.json")
	if err := f.writeFirecrackerConfig(configPath, fcConfig); err != nil {
		f.cleanup(vmSockDir)
		return nil, fmt.Errorf("write firecracker config: %w", err)
	}

	// Step 4: Start Firecracker process
	cmd := exec.CommandContext(ctx, f.binaryPath, "--config-file", configPath)

	// Optionally capture stdout/stderr for debugging
	logPath := filepath.Join(vmSockDir, "firecracker.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		f.cleanup(vmSockDir)
		return nil, fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		f.cleanup(vmSockDir)
		return nil, fmt.Errorf("start firecracker process: %w", err)
	}

	pid := cmd.Process.Pid
	f.logger.InfoContext(ctx, "firecracker process started",
		"id", id,
		"pid", pid)

	// Step 5: Wait for socket to appear (healthcheck)
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	if err := f.waitForSocket(ctx, socketPath, timeout); err != nil {
		f.cleanup(vmSockDir)
		return nil, fmt.Errorf("firecracker healthcheck failed: %w", err)
	}

	f.logger.InfoContext(ctx, "firecracker vm started successfully",
		"id", id,
		"pid", pid,
		"socket", socketPath)

	// Return VMInstance
	return &VMInstance{
		ID:         id,
		AppID:      config.AppID,
		PID:        pid,
		SocketPath: socketPath,
		Meta:       make(map[string]interface{}),
		StartedAt:  time.Now(),
	}, nil
}

// Stop terminates a running Firecracker VM.
// Currently performs process cleanup; future implementation will use Firecracker API.
func (f *firecracker) Stop(ctx context.Context, instance *VMInstance) error {
	f.logger.InfoContext(ctx, "stopping firecracker vm", "id", instance.ID)

	// TODO: Send shutdown command via Firecracker API socket
	// For now, we just clean up the socket directory

	// Clean up socket directory
	vmSockDir := filepath.Dir(instance.SocketPath)
	if err := f.cleanup(vmSockDir); err != nil {
		f.logger.WarnContext(ctx, "failed to cleanup socket directory", "error", err)
		return err
	}

	f.logger.InfoContext(ctx, "firecracker vm stopped", "id", instance.ID)
	return nil
}

// Status checks the current state of a VM.
// Returns VMStatusRunning if the process is still alive, VMStatusStopped otherwise.
func (f *firecracker) Status(ctx context.Context, instance *VMInstance) (VMStatus, error) {
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

// --- Private Helper Methods ---

// validateConfig checks that all required paths exist and are valid.
// Validates all three block devices: rootfs (pre-built, shared), appfs (built from OCI), and statefs (empty writable).
func (f *firecracker) validateConfig(config VMConfig) error {
	if _, err := os.Stat(config.RootFsPath); err != nil {
		return fmt.Errorf("rootfs not found at %s: %w", config.RootFsPath, err)
	}
	if _, err := os.Stat(config.AppFsPath); err != nil {
		return fmt.Errorf("appfs not found at %s: %w", config.AppFsPath, err)
	}
	if _, err := os.Stat(config.KernelPath); err != nil {
		return fmt.Errorf("kernel not found at %s: %w", config.KernelPath, err)
	}
	if config.VCPU <= 0 {
		config.VCPU = 1
	}
	if config.Memory <= 0 {
		config.Memory = 512
	}
	return nil
}

// buildFirecrackerConfig creates the Firecracker JSON configuration.
// Configures three block devices in order:
//  1. rootfs (root device, read-only, pre-built) - /var/lib/walkio/base/[version]/rootfs.ext4
//     Shared across multiple VMs, contains system initialization and boot scripts.
//  2. app (secondary device, read-only) - /var/lib/walkio/apps/[digest].ext4
//     Built from OCI image layers, contains application code and data.
//  3. state (secondary device, writable) - /var/lib/walkio/state/[vm-uuid].ext4
//     Empty ext4 filesystem, writable layer for runtime state (logs, temp files, etc).
func (f *firecracker) buildFirecrackerConfig(config VMConfig, socketPath string) map[string]interface{} {
	return map[string]interface{}{
		"vm_config": map[string]interface{}{
			"vcpu_count":   config.VCPU,
			"mem_size_mib": config.Memory,
		},
		"kernel": map[string]interface{}{
			"kernel_image_path": config.KernelPath,
		},
		"drives": []map[string]interface{}{
			// Drive 1: RootFS - system initialization (root device, read-only, shared)
			{
				"drive_id":       "rootfs",
				"path_on_host":   config.RootFsPath,
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
				"path_on_host":   config.StateFsPath,
				"is_root_device": false,
				"is_read_only":   false,
			},
		},
		"ioapic": map[string]interface{}{
			"enabled": true,
		},
		"socket_path": socketPath,
	}
}

// writeFirecrackerConfig writes the configuration to a JSON file.
func (f *firecracker) writeFirecrackerConfig(path string, config map[string]interface{}) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// waitForSocket polls for the socket file to appear within the given timeout.
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

// cleanup removes the VM socket directory.
func (f *firecracker) cleanup(vmSockDir string) error {
	return os.RemoveAll(vmSockDir)
}

// generateVMID creates a unique VM identifier.
// Currently uses UUID v4 format as per the design.
func generateVMID() (string, error) {
	// Use UUID for consistency with StateFS ID format
	// Import uuid package if not already imported
	return generateUUID()
}

// generateUUID generates a UUID v4 string.
// Simple implementation - in production should use github.com/google/uuid.
func generateUUID() (string, error) {
	// Create a pseudo-UUID from timestamp and random
	// This is a placeholder - actual implementation should use google/uuid
	t := time.Now().UnixNano()
	r := rand.Int63()
	return fmt.Sprintf("%016x-%04x-4%03x-%04x-%012x",
		t>>32, (t>>16)&0xffff, t&0xfff,
		(r>>48)&0x3fff, r&0xffffffffffff), nil
}
