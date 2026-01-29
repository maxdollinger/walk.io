package oci

import (
	"context"
)

// OciImageSource abstracts where images come from (registry, local, tar, etc.)
type OciImageSource interface {
	GetImage(ctx context.Context) (*Image, error)
	Info() string
}
