package runtime

import "time"

// VMConfig holds essential Firecracker VM configuration.
// This is intentionally minimal to keep the design clean and extensible.
type VMConfig struct {
	AppID       string        // which app this VM is running
	RootFsPath  string        // path to /var/lib/walkio/base/[version]/rootfs.ext4 (pre-built, shared)
	AppFsPath   string        // path to /var/lib/walkio/apps/{digest}.ext4
	StateFsPath string        // path to /var/lib/walkio/state/{uuid}.ext4
	KernelPath  string        // path to firecracker kernel from base bundle
	BaseVersion string        // base bundle version (e.g., "v1.0") for reference/logging
	VCPU        int           // number of vCPUs (default: 1)
	Memory      int           // memory in MB (default: 512)
	Timeout     time.Duration // operation timeout
}

// VMInstance represents a running Firecracker VM instance (a Crutch).
type VMInstance struct {
	ID         string                 // UUID of this VM instance
	AppID      string                 // which app is running
	PID        int                    // firecracker process PID
	SocketPath string                 // firecracker control socket path
	Meta       map[string]interface{} // extensible metadata for future features (networking, etc.)
	StartedAt  time.Time
}

// GetStateFsPath computes the state filesystem path from the VM instance ID.
// This ensures state_fs_path always matches the VM instance UUID.
// Returns: /var/lib/walkio/state/{id}.ext4
func (v *VMInstance) GetStateFsPath() string {
	return "/var/lib/walkio/state/" + v.ID + ".ext4"
}

// VMStatus represents the current operational state of a VM.
type VMStatus string

const (
	VMStatusRunning VMStatus = "running"
	VMStatusStopped VMStatus = "stopped"
	VMStatusError   VMStatus = "error"
)
