package fs

import (
	"context"

	"github.com/maxdollinger/walk.io/pkg/oci"
)

type AppConfigInjector interface {
	// Prepare injects /walk/argv and /walk/env into the rootfs
	InjectAppConfig(ctx context.Context, rootfsDir string, config *oci.ImageConfig) error
}

type NoOpFilesystemPreparer struct{}

func NewNoOpFilesystemPreparer() *NoOpFilesystemPreparer {
	return &NoOpFilesystemPreparer{}
}

func (p *NoOpFilesystemPreparer) InjectAppConfig(ctx context.Context, rootfsDir string, config *oci.ImageConfig) error {
	// No-op: in real implementation, would create /walk/argv and /walk/env
	return nil
}
