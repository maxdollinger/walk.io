package oci

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/go-digest"
)

// RegistryProvider fetches OCI images from a container registry using go-containerregistry.
// It implements the ImageProvider interface.
//
// Image references need to be fully quallified like docker.io/libray/nginx:latest
//
// Once created, GetImage() downloads the image manifest, config, and layer metadata
// from the registry. The actual layer content is not downloaded until Extract() is called.
type RegistryProvider struct {
	imageRef name.Reference // e.g., "nginx:latest" or "docker.io/nginx:latest"
}

// NewRegistryProvider creates a new provider for the given image reference
// ref can be:
//   - "nginx:latest" (defaults to docker.io/library)
//   - "docker.io/nginx:latest"
//   - "ghcr.io/owner/repo:tag"
//   - "localhost:5000/image:tag"
func NewRegistryProvider(imageRef string) (OciImageSource, error) {
	// Add docker.io default if no registry specified
	normalizedRef := imageRef
	if !strings.Contains(imageRef, "/") {
		normalizedRef = "docker.io/library/" + imageRef
	} else if !strings.Contains(strings.Split(imageRef, "/")[0], ".") && !strings.Contains(strings.Split(imageRef, "/")[0], ":") {
		// If first component has no dots or colons, prepend docker.io
		normalizedRef = "docker.io/" + imageRef
	}

	ref, err := name.ParseReference(normalizedRef)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference: %w", err)
	}

	return &RegistryProvider{
		imageRef: ref,
	}, nil
}

func (p *RegistryProvider) Info() string {
	return p.imageRef.String()
}

// GetImage fetches the image from the registry and returns an Image with all layers
func (p *RegistryProvider) GetImage(ctx context.Context) (*Image, error) {
	platformStr := fmt.Sprintf("linux/%s", runtime.GOARCH)
	platform, err := v1.ParsePlatform(platformStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse platform: %w", err)
	}

	// Fetch the image from the registry
	img, err := remote.Image(p.imageRef, remote.WithContext(ctx), remote.WithPlatform(*platform))
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}

	// Get the image digest (for cache key)
	dgst, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("get image digest: %w", err)
	}

	// Get the manifest
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("get manifest: %w", err)
	}

	// Parse the image config
	config, err := parseImageConfig(img)
	if err != nil {
		return nil, fmt.Errorf("parse image config: %w", err)
	}

	// Get the layers
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("get layers: %w", err)
	}

	// Wrap layers with our Layer interface
	wrappedLayers := make([]Layer, len(layers))
	for i, layer := range layers {
		wrappedLayers[i] = &registryLayer{layer: layer}
	}

	// Calculate manifest size from config descriptor
	manifestSize := manifest.Config.Size
	for _, layer := range manifest.Layers {
		manifestSize += layer.Size
	}

	return &Image{
		Digest: digest.Digest(dgst.String()),
		Config: config,
		Layers: wrappedLayers,
		Manifest: &Manifest{
			MediaType: string(manifest.MediaType),
			Size:      manifestSize,
		},
	}, nil
}

// parseImageConfig extracts the OCI config from the image
func parseImageConfig(img v1.Image) (*ImageConfig, error) {
	cfgFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("get config file: %w", err)
	}

	if cfgFile == nil {
		return nil, fmt.Errorf("no config file in image")
	}

	cfg := cfgFile.Config

	return &ImageConfig{
		Entrypoint: cfg.Entrypoint,
		Cmd:        cfg.Cmd,
		Env:        cfg.Env,
		WorkingDir: cfg.WorkingDir,
		User:       cfg.User,
	}, nil
}

// registryLayer wraps a go-containerregistry layer to implement the Layer interface.
// It provides lazy access to layer content - data is only downloaded when Extract() is called.
type registryLayer struct {
	layer v1.Layer
}

func (l *registryLayer) Digest() digest.Digest {
	dgst, err := l.layer.Digest()
	if err != nil {
		return digest.Digest("")
	}
	// Convert go-containerregistry digest to opencontainers digest
	return digest.Digest(dgst.String())
}

func (l *registryLayer) Size() int64 {
	size, err := l.layer.Size()
	if err != nil {
		return 0
	}
	return size
}

func (l *registryLayer) MediaType() string {
	mediaType, err := l.layer.MediaType()
	if err != nil {
		return ""
	}
	return string(mediaType)
}

// Compressed returns a reader for the compressed (tar.gz) layer data
func (l *registryLayer) Compressed(ctx context.Context) (io.ReadCloser, error) {
	reader, err := l.layer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("get compressed layer: %w", err)
	}
	return reader, nil
}

// NoOpImageProvider for testing
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
