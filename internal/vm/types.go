package vm

import (
	"path"
	"time"
)

const WALKIO_PATH = "/var/lib/walkio/"

// VMConfig holds essential Firecracker VM configuration.
// This is intentionally minimal to keep the design clean and extensible.
type VMConfig struct {
	AppID       string        // which app this VM is running
	AppFsPath   string        // path to /var/lib/walkio/apps/{digest}.ext4
	BaseVersion string        // base bundle version (e.g., "v1.0") for reference/logging
	VCPU        int           // number of vCPUs (default: 1)
	Memory      int           // memory in MB (default: 512)
	Timeout     time.Duration // operation timeout
}

func (c *VMConfig) GetRootFSPath() string {
	return path.Join(WALKIO_PATH, "base", c.BaseVersion, "rootfs.ext4")
}

func (c *VMConfig) GetKernelPath() string {
	return path.Join(WALKIO_PATH, "base", c.BaseVersion, "vmlinux")
}

func (c *VMConfig) GetFirecrackerPath() string {
	return path.Join(WALKIO_PATH, "base", c.BaseVersion, "firecracker")
}

// VMInstance represents a running Firecracker VM instance (a Crutch).
type VMInstance struct {
	ID           string // UUID of this VM instance
	PID          int    // firecracker process PID
	SocketPath   string // firecracker control socket path
	ConfigPath   string
	LogPath      string
	StateDevPath string
	VMConfig     *VMConfig
	Meta         map[string]any // extensible metadata for future features (networking, etc.)
	StartedAt    time.Time
}

// VMStatus represents the current operational state of a VM.
type VMStatus string

const (
	VMStatusRunning VMStatus = "running"
	VMStatusStopped VMStatus = "stopped"
	VMStatusError   VMStatus = "error"
)
