package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxdollinger/walk.io/pkg/fs"
	"github.com/maxdollinger/walk.io/pkg/lock"
	"github.com/maxdollinger/walk.io/pkg/oci"
)

// TestBuilderWiring verifies that all components are correctly wired together
func TestBuilderWiring(t *testing.T) {
	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "walk-builder-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create builder with all no-op implementations
	builder := NewBuilder(
		fs.NewNoOpLayerFlattener(),
		fs.NewNoOpFilesystemPreparer(),
		fs.NewNoOpBlockDeviceBuilder(),
		lock.NewNoOpLocker(),
	)

	// Create a dummy registry source
	source := oci.NewNoOpImageProvider()

	// Build
	ctx := context.Background()
	result, err := builder.Build(ctx, source, BuildOptions{
		OutputDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("result is nil")
	}

	if result.SourceDigest.String() == "" {
		t.Error("source digest is empty")
	}

	if result.ImageConfig == nil {
		t.Error("image config is nil")
	}

	if result.BlockDevicePath == "" {
		t.Error("block device path is empty")
	}

	expectedPath := filepath.Join(tmpDir, result.SourceDigest.Hex()+".ext4")
	if result.BlockDevicePath != expectedPath {
		t.Errorf("unexpected block device path: got %s, want %s", result.BlockDevicePath, expectedPath)
	}

	if result.BuildTime == 0 {
		t.Error("build time is zero")
	}

	t.Logf("Build result: digest=%s, path=%s, cached=%v, time=%v",
		result.SourceDigest,
		result.BlockDevicePath,
		result.Cached,
		result.BuildTime)
}

// TestBuilderCaching verifies that cached builds are properly detected
func TestBuilderCaching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "walk-builder-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	builder := NewBuilder(
		fs.NewNoOpLayerFlattener(),
		fs.NewNoOpFilesystemPreparer(),
		fs.NewNoOpBlockDeviceBuilder(),
		lock.NewNoOpLocker(),
	)

	provider := oci.NewNoOpImageProvider()
	ctx := context.Background()

	// First build
	result1, err := builder.Build(ctx, provider, BuildOptions{
		OutputDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("first build failed: %v", err)
	}

	if result1.Cached {
		t.Error("first build should not be cached")
	}

	// Create the output file to simulate a completed build
	// Note: In no-op mode, the file isn't actually created, so we create it manually
	if err := os.WriteFile(result1.BlockDevicePath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to create dummy output file: %v", err)
	}

	// Second build (should be cached)
	result2, err := builder.Build(ctx, provider, BuildOptions{
		OutputDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("second build failed: %v", err)
	}

	if !result2.Cached {
		t.Error("second build should be cached")
	}

	if result1.SourceDigest != result2.SourceDigest {
		t.Error("digests should match between builds")
	}

	if result1.BlockDevicePath != result2.BlockDevicePath {
		t.Error("block device paths should match between builds")
	}

	t.Logf("First build: cached=%v, time=%v", result1.Cached, result1.BuildTime)
	t.Logf("Second build: cached=%v, time=%v", result2.Cached, result2.BuildTime)
}
