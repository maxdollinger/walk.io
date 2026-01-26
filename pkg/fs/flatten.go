package fs

import (
	"context"

	"github.com/maxdollinger/walk.io/pkg/oci"
)

type FsBuilder interface {
	// Flatten extracts all layers to target directory, handling whiteouts
	BuildFs(ctx context.Context, layers []oci.Layer, targetDir string) error
}

// NoOpLayerFlattener is a no-op implementation for testing
type NoOpLayerFlattener struct{}

// NewNoOpLayerFlattener creates a new no-op layer flattener
func NewNoOpLayerFlattener() *NoOpLayerFlattener {
	return &NoOpLayerFlattener{}
}

func (f *NoOpLayerFlattener) BuildFs(ctx context.Context, layers []oci.Layer, targetDir string) error {
	// No-op: in real implementation, would extract and merge layers
	return nil
}
