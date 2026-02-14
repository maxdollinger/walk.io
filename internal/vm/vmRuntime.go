package vm

import "context"

type VMRuntime interface {
	// Start launches a Firecracker microVM with the given configuration.
	// Returns a VMInstance on success, or an error on failure.
	Start(ctx context.Context, config VMConfig, stateDevPath string) (*VMInstance, error)

	// Stop terminates a running Firecracker VM.
	Stop(ctx context.Context, instance *VMInstance) error

	// Status checks the current state of a VM.
	Status(ctx context.Context, instance *VMInstance) (VMStatus, error)

	// Future: Pause, Resume, GetMetrics
}
