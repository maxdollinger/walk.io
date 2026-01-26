package fs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxdollinger/walk.io/pkg/oci"
	"github.com/opencontainers/go-digest"
)

// mockLayer creates a mock OCI layer with specified content
type mockLayer struct {
	digest   digest.Digest
	size     int64
	contents []tarEntry
}

type tarEntry struct {
	name     string
	typeflag byte
	content  []byte
	linkname string
	mode     int64
}

// newMockLayer creates a mock layer from tar entries
func newMockLayer(entries ...tarEntry) *mockLayer {
	// Create a tar.gz in memory
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Typeflag: entry.typeflag,
			Size:     int64(len(entry.content)),
			Mode:     entry.mode,
			Linkname: entry.linkname,
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			panic(err)
		}

		if len(entry.content) > 0 {
			if _, err := tarWriter.Write(entry.content); err != nil {
				panic(err)
			}
		}
	}

	tarWriter.Close()
	gzipWriter.Close()

	return &mockLayer{
		digest:   digest.FromString("mock"),
		size:     int64(buf.Len()),
		contents: entries,
	}
}

func (l *mockLayer) Digest() digest.Digest {
	return l.digest
}

func (l *mockLayer) Size() int64 {
	return l.size
}

func (l *mockLayer) MediaType() string {
	return "application/vnd.docker.image.rootfs.diff.tar.gzip"
}

func (l *mockLayer) Compressed(ctx context.Context) (io.ReadCloser, error) {
	// Recreate the tar.gz
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range l.contents {
		header := &tar.Header{
			Name:     entry.name,
			Typeflag: entry.typeflag,
			Size:     int64(len(entry.content)),
			Mode:     entry.mode,
			Linkname: entry.linkname,
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}

		if len(entry.content) > 0 {
			if _, err := tarWriter.Write(entry.content); err != nil {
				return nil, err
			}
		}
	}

	tarWriter.Close()
	gzipWriter.Close()

	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// TestLayerFlattenerBasicExtraction tests extracting a simple layer
func TestLayerFlattenerBasicExtraction(t *testing.T) {
	tmpDir := t.TempDir()

	layer := newMockLayer(
		tarEntry{name: "file.txt", typeflag: tar.TypeReg, content: []byte("hello"), mode: 0o644},
		tarEntry{name: "dir/", typeflag: tar.TypeDir, mode: 0o755},
		tarEntry{name: "dir/nested.txt", typeflag: tar.TypeReg, content: []byte("world"), mode: 0o644},
	)

	flattener := NewLayerFlattener()
	err := flattener.BuildFs(context.Background(), []oci.Layer{layer}, tmpDir)
	if err != nil {
		t.Fatalf("BuildFs failed: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(tmpDir, "file.txt")); err != nil {
		t.Errorf("file.txt not extracted: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "dir")); err != nil {
		t.Errorf("dir/ not extracted: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "dir", "nested.txt")); err != nil {
		t.Errorf("dir/nested.txt not extracted: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(filepath.Join(tmpDir, "file.txt"))
	if err != nil {
		t.Fatalf("read file.txt: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("file.txt content = %q, want %q", string(content), "hello")
	}
}

// TestLayerFlattenerLayerOverwrite tests that later layers overwrite earlier ones
func TestLayerFlattenerLayerOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	layer1 := newMockLayer(
		tarEntry{name: "file.txt", typeflag: tar.TypeReg, content: []byte("original"), mode: 0o644},
	)

	layer2 := newMockLayer(
		tarEntry{name: "file.txt", typeflag: tar.TypeReg, content: []byte("updated"), mode: 0o644},
	)

	flattener := NewLayerFlattener()
	err := flattener.BuildFs(context.Background(), []oci.Layer{layer1, layer2}, tmpDir)
	if err != nil {
		t.Fatalf("BuildFs failed: %v", err)
	}

	// Verify content is from layer2
	content, err := os.ReadFile(filepath.Join(tmpDir, "file.txt"))
	if err != nil {
		t.Fatalf("read file.txt: %v", err)
	}
	if string(content) != "updated" {
		t.Errorf("file.txt content = %q, want %q", string(content), "updated")
	}
}

// TestLayerFlattenerWhiteout tests OCI whiteout handling
func TestLayerFlattenerWhiteout(t *testing.T) {
	tmpDir := t.TempDir()

	layer1 := newMockLayer(
		tarEntry{name: "file.txt", typeflag: tar.TypeReg, content: []byte("delete me"), mode: 0o644},
	)

	layer2 := newMockLayer(
		// .wh.file.txt indicates that file.txt should be deleted
		tarEntry{name: ".wh.file.txt", typeflag: tar.TypeReg, mode: 0o644},
	)

	flattener := NewLayerFlattener()
	err := flattener.BuildFs(context.Background(), []oci.Layer{layer1, layer2}, tmpDir)
	if err != nil {
		t.Fatalf("BuildFs failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filepath.Join(tmpDir, "file.txt")); !os.IsNotExist(err) {
		t.Errorf("file.txt should have been deleted by whiteout")
	}
}

// TestLayerFlattenerOpaqueWhiteout tests opaque whiteout handling
func TestLayerFlattenerOpaqueWhiteout(t *testing.T) {
	tmpDir := t.TempDir()

	layer1 := newMockLayer(
		tarEntry{name: "dir/", typeflag: tar.TypeDir, mode: 0o755},
		tarEntry{name: "dir/file1.txt", typeflag: tar.TypeReg, content: []byte("file1"), mode: 0o644},
		tarEntry{name: "dir/file2.txt", typeflag: tar.TypeReg, content: []byte("file2"), mode: 0o644},
	)

	layer2 := newMockLayer(
		// .wh..wh..opaque indicates that the entire dir/ should be emptied
		tarEntry{name: "dir/.wh..wh..opaque", typeflag: tar.TypeReg, mode: 0o644},
		tarEntry{name: "dir/newfile.txt", typeflag: tar.TypeReg, content: []byte("new"), mode: 0o644},
	)

	flattener := NewLayerFlattener()
	err := flattener.BuildFs(context.Background(), []oci.Layer{layer1, layer2}, tmpDir)
	if err != nil {
		t.Fatalf("BuildFs failed: %v", err)
	}

	// Verify old files were deleted
	if _, err := os.Stat(filepath.Join(tmpDir, "dir", "file1.txt")); !os.IsNotExist(err) {
		t.Errorf("dir/file1.txt should have been deleted by opaque whiteout")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "dir", "file2.txt")); !os.IsNotExist(err) {
		t.Errorf("dir/file2.txt should have been deleted by opaque whiteout")
	}

	// Verify new file exists
	if _, err := os.Stat(filepath.Join(tmpDir, "dir", "newfile.txt")); err != nil {
		t.Errorf("dir/newfile.txt should exist: %v", err)
	}
}

// TestLayerFlattenerMultipleLayers tests merging multiple layers
func TestLayerFlattenerMultipleLayers(t *testing.T) {
	tmpDir := t.TempDir()

	layer1 := newMockLayer(
		tarEntry{name: "file1.txt", typeflag: tar.TypeReg, content: []byte("layer1"), mode: 0o644},
	)

	layer2 := newMockLayer(
		tarEntry{name: "file2.txt", typeflag: tar.TypeReg, content: []byte("layer2"), mode: 0o644},
	)

	layer3 := newMockLayer(
		tarEntry{name: "file3.txt", typeflag: tar.TypeReg, content: []byte("layer3"), mode: 0o644},
	)

	flattener := NewLayerFlattener()
	err := flattener.BuildFs(context.Background(), []oci.Layer{layer1, layer2, layer3}, tmpDir)
	if err != nil {
		t.Fatalf("BuildFs failed: %v", err)
	}

	// Verify all files exist
	for i, name := range []string{"file1.txt", "file2.txt", "file3.txt"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err != nil {
			t.Errorf("%s not extracted: %v", name, err)
		}

		content, _ := os.ReadFile(filepath.Join(tmpDir, name))
		expected := []byte("layer" + string(rune('1'+i)))
		if !bytes.Equal(content, expected) {
			t.Errorf("%s content = %q, want %q", name, string(content), string(expected))
		}
	}
}

// TestLayerFlattenerContextCancellation tests that context cancellation works
func TestLayerFlattenerContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	layer := newMockLayer(
		tarEntry{name: "file.txt", typeflag: tar.TypeReg, content: []byte("content"), mode: 0o644},
	)

	flattener := NewLayerFlattener()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := flattener.BuildFs(ctx, []oci.Layer{layer}, tmpDir)
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
}
