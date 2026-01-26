package oci

import (
	"github.com/opencontainers/go-digest"
)

// Image represents an OCI image with its metadata and layers
type Image struct {
	Digest   digest.Digest
	Config   *ImageConfig
	Layers   []Layer
	Manifest *Manifest
}

// ImageConfig contains OCI runtime configuration
type ImageConfig struct {
	Entrypoint []string
	Cmd        []string
	Env        []string
	WorkingDir string
	User       string
}

// Manifest represents the OCI manifest
type Manifest struct {
	MediaType string
	Size      int64
}
