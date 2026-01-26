package oci

import (
	"context"

	"github.com/opencontainers/go-digest"
)

// Layer represents a single OCI layer
type Layer interface {
	Digest() digest.Digest
	Size() int64
	MediaType() string
	// Extract writes the layer contents to the target directory
	Extract(ctx context.Context, target string) error
}
