package oci

import (
	"context"
)

// ImageProvider abstracts where images come from (registry, local, tar, etc.)
type ImageProvider interface {
	GetImage(ctx context.Context) (*Image, error)
	Info() string
}
