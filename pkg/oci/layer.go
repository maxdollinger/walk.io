package oci

import (
	"context"
	"io"

	"github.com/opencontainers/go-digest"
)

// Layer represents a single OCI layer
type Layer interface {
	Digest() digest.Digest
	Size() int64
	MediaType() string
	// Compressed returns a reader for the compressed (tar.gz) layer data
	// The caller must close the reader when done
	Compressed(ctx context.Context) (io.ReadCloser, error)
}
