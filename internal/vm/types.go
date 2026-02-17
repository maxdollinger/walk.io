package vm

import (
	"path"
	"time"
)

const WALKIO_PATH = "/var/lib/walkio/"

// ExposedPort represents a container port exposed by the OCI image
type ExposedPort struct {
	Port     int    // Port number
	Protocol string // Protocol: "tcp" or "udp"
}

// VMConfig holds essential Firecracker VM configuration.
// This is intentionally minimal to keep the design clean and extensible.
type VMConfig struct {
	AppID       string        // which app this VM is running
	AppFsPath   string        // path to /var/lib/walkio/apps/{digest}.ext4
	BaseVersion string        // base bundle version (e.g., "v1.0") for reference/logging
	VCPU        int           // number of vCPUs (default: 1)
	Memory      int           // memory in MB (default: 512)
	Timeout     time.Duration // operation timeout

	// Network configuration (default: true)
	NetworkEnabled bool          // Whether to setup networking for this VM
	ExposedPorts   []ExposedPort // Ports exposed by the OCI image
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

// VMStatus represents the current operational state of a VM.
type VMStatus string

const (
	VMStatusRunning VMStatus = "running"
	VMStatusStopped VMStatus = "stopped"
	VMStatusError   VMStatus = "error"
)
