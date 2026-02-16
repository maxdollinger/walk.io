package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/maxdollinger/walk.io/pkg/utils"
)

const (
	LOG_DIR = "/var/walkio/machines/logs"
	VM_DIR  = "/var/walkio/machines/"
)

type FirecrackerMachine struct {
	ID            string
	Cmd           *exec.Cmd
	LogFile       *os.File
	SocketPath    string
	ConfigPath    string
	MachineConfig *VMConfig
}

func NewFirecrackerMachine(stateDevPath string, config *VMConfig) (*FirecrackerMachine, error) {
	id, err := utils.NewUUID7()
	if err != nil {
		return nil, fmt.Errorf("generate vm id: %w", err)
	}

	machineDir := path.Join(VM_DIR, id)
	if err := os.MkdirAll(machineDir, 0o755); err != nil {
		return nil, fmt.Errorf("could not create machineDir: %w", err)
	}

	fcConfig := buildFirecrackerConfig(config, stateDevPath)
	data, err := json.Marshal(fcConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	configPath := filepath.Join(machineDir, id+".json")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("write config file: %w", err)
	}

	socketPath := filepath.Join(machineDir, id+".sock")
	logPath := filepath.Join(LOG_DIR, id+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		err = errors.Join(err, os.RemoveAll(machineDir))
		return nil, fmt.Errorf("could not create log file: %w", err)
	}

	instance := FirecrackerMachine{
		ID:            id,
		Cmd:           nil,
		SocketPath:    socketPath,
		LogFile:       logFile,
		ConfigPath:    configPath,
		MachineConfig: config,
	}

	return &instance, nil
}

func (m *FirecrackerMachine) Start() error {
	_ = os.Remove(m.SocketPath)

	cmd := exec.Command(m.MachineConfig.GetFirecrackerPath(), "--api-sock", m.SocketPath, "--config-file", m.ConfigPath)
	cmd.Stdout = m.LogFile
	cmd.Stderr = m.LogFile
	if err := cmd.Start(); err != nil {
		err = errors.Join(err, m.Clean())
		return fmt.Errorf("start firecracker process: %w", err)
	}

	return nil
}

func (m *FirecrackerMachine) Status() (VMStatus, error) {
	if m.Cmd == nil {
		return VMStatusStopped, nil
	}

	// Try to send signal 0 to check if process is alive
	if err := m.Cmd.Process.Signal(os.Signal(nil)); err != nil {
		return VMStatusStopped, nil
	}

	return VMStatusRunning, nil
}

func (m *FirecrackerMachine) Stop() error {
	if m.Cmd == nil {
		return nil
	}

	_ = m.Cmd.Process.Kill()
	err := m.Cmd.Wait()
	if err != nil {
		return err
	}

	err = os.Remove(m.SocketPath)
	if err != nil {
		return err
	}
	m.Cmd = nil
	return nil
}

func (m *FirecrackerMachine) Clean() error {
	if m.Cmd != nil {
		return fmt.Errorf("machine %s is still running", m.ID)
	}

	err := os.RemoveAll(path.Join(VM_DIR, m.ID))
	if err != nil {
		return fmt.Errorf("could not clean vm %s: %w", m.ID, err)
	}

	_ = m.LogFile.Close()

	m.ConfigPath = ""
	m.SocketPath = ""

	return nil
}

func buildFirecrackerConfig(config *VMConfig, stateDevPath string) map[string]any {
	return map[string]any{
		"boot-source": map[string]any{
			"kernel_image_path": config.GetKernelPath(),
			"boot_args":         "console=ttyS0 reboot=k panic=1 init=/walkio/init",
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
