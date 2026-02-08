package statefs

import "time"

// StateFSInstance represents a created StateFS block device.
type StateFSInstance struct {
	ID              string // UUID of this StateFS
	AppID           string // which app this StateFS belongs to
	BlockDevicePath string // path to .ext4 file
	SizeBytes       int64  // size of block device in bytes
	Persistent      bool   // persistent or ephemeral (for future use)
	CreatedAt       time.Time
}

// StateFSBuildOptions specifies parameters for building a StateFS block device.
type StateFSBuildOptions struct {
	SizeBytes int64  // size in bytes of empty block device
	OutputDir string // directory to place .ext4 file
	WorkDir   string // temporary build directory
}
