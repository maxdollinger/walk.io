package oci

import (
	"context"

	"github.com/opencontainers/go-digest"
)

type NoOpImageProvider struct{}

func NewNoOpImageProvider() *NoOpImageProvider {
	return &NoOpImageProvider{}
}

func (p *NoOpImageProvider) Info() string {
	return "registry.com/noop-image:latest"
}

func (p *NoOpImageProvider) GetImage(ctx context.Context) (*Image, error) {
	// Return a dummy image with a fake digest
	return &Image{
		Digest: digest.FromString("noop-image"),
		Config: &ImageConfig{
			Entrypoint: []string{"/bin/sh"},
			Cmd:        []string{"-c", "echo hello"},
			Env:        []string{"PATH=/usr/bin:/bin"},
			WorkingDir: "/",
			User:       "root",
		},
		Layers:   []Layer{},
		Manifest: &Manifest{MediaType: "application/vnd.oci.image.manifest.v1+json"},
	}, nil
}
