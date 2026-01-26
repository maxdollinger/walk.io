package oci

import (
	"context"
	"testing"
)

func TestNewRegistryProvider(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple image name defaults to docker.io",
			input: "nginx",
			want:  "docker.io/library/nginx",
		},
		{
			name:  "image with tag defaults to docker.io",
			input: "nginx:1.21",
			want:  "docker.io/library/nginx:1.21",
		},
		{
			name:  "full reference with docker.io",
			input: "docker.io/library/nginx:latest",
			want:  "docker.io/library/nginx:latest",
		},
		{
			name:  "ghcr reference",
			input: "ghcr.io/owner/repo:v1.0",
			want:  "ghcr.io/owner/repo:v1.0",
		},
		{
			name:  "localhost registry",
			input: "localhost:5000/myimage:latest",
			want:  "localhost:5000/myimage:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewRegistryProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRegistryProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			got := provider.Info()
			if got != tt.want {
				t.Errorf("Info() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRegistryProviderInfo(t *testing.T) {
	provider, err := NewRegistryProvider("busybox")
	if err != nil {
		t.Fatalf("NewRegistryProvider failed: %v", err)
	}

	info := provider.Info()
	if info == "" {
		t.Error("Info() returned empty string")
	}

	if !contains(info, "busybox") {
		t.Errorf("Info() = %q, should contain 'busybox'", info)
	}
}

func TestNoOpImageProvider(t *testing.T) {
	provider := NewNoOpImageProvider()

	info := provider.Info()
	if info == "" {
		t.Error("Info() returned empty string")
	}

	image, err := provider.GetImage(context.Background())
	if err != nil {
		t.Fatalf("GetImage failed: %v", err)
	}

	if image == nil {
		t.Fatal("GetImage returned nil image")
	}

	if image.Config == nil {
		t.Fatal("GetImage returned image with nil config")
	}

	if image.Manifest == nil {
		t.Fatal("GetImage returned image with nil manifest")
	}

	if len(image.Config.Entrypoint) == 0 {
		t.Error("image config has no entrypoint")
	}
}

// contains checks if needle is in haystack
func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
