// Package fs provides filesystem operations for building container rootfs images.
//
// The main component is the LayerFlattener, which extracts and merges OCI image
// layers into a single filesystem representation. It correctly handles:
//   - Layer ordering and file overwrites
//   - OCI whiteout markers (.wh.* files) for deletions
//   - Opaque whiteouts (.wh..wh..opaque) for directory clearing
//   - Directory traversal protection
//   - Context cancellation
//
// The package also provides interfaces for injecting application metadata
// and building block device images from the prepared filesystem.
package fs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxdollinger/walk.io/pkg/oci"
)

type FsBuilder interface {
	BuildFs(ctx context.Context, layers []oci.Layer, targetDir string) error
}

type LayerFlattener struct{}

func NewLayerFlattener() *LayerFlattener {
	return &LayerFlattener{}
}

func (f *LayerFlattener) BuildFs(ctx context.Context, layers []oci.Layer, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	for i, layer := range layers {
		if err := f.extractLayer(ctx, layer, targetDir); err != nil {
			return fmt.Errorf("extract layer %d: %w", i, err)
		}
	}

	return nil
}

func (f *LayerFlattener) extractLayer(ctx context.Context, layer oci.Layer, targetDir string) error {
	reader, err := layer.Compressed(ctx)
	if err != nil {
		return fmt.Errorf("get compressed layer: %w", err)
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("decompress gzip: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar header: %w", err)
		}

		if isWhiteout(header.Name) {
			if err := f.handleWhiteout(targetDir, header.Name); err != nil {
				return fmt.Errorf("handle whiteout: %w", err)
			}
			continue
		}

		if err := f.extractTarEntry(targetDir, header, tarReader); err != nil {
			return fmt.Errorf("extract tar entry %q: %w", header.Name, err)
		}

	}

	return nil
}

func isWhiteout(name string) bool {
	// OCI whiteout: .wh.FILENAME deletes FILENAME
	// Opaque whiteout: .wh..wh..opaque deletes the directory
	_, file := filepath.Split(filepath.Clean(name))
	return strings.HasPrefix(file, ".wh.")
}

// handleWhiteout removes a file or directory indicated by a whiteout marker
func (f *LayerFlattener) handleWhiteout(targetDir, whiteoutPath string) error {
	// Remove .wh. prefix to get the actual filename
	dir, file := filepath.Split(filepath.Clean(whiteoutPath))
	actualName := strings.TrimPrefix(file, ".wh.")

	// Reconstruct the full path of what to delete
	deletePath := filepath.Join(targetDir, dir, actualName)

	// Check for opaque whiteout
	if actualName == ".wh..opaque" {
		// Remove the entire directory
		opaqueDir := filepath.Join(targetDir, dir)
		if err := os.RemoveAll(opaqueDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove opaque directory: %w", err)
		}
		// Recreate the directory (it should still exist, just be empty)
		if err := os.MkdirAll(opaqueDir, 0o755); err != nil {
			return fmt.Errorf("recreate opaque directory: %w", err)
		}
		return nil
	}

	// Regular whiteout: delete the file
	if err := os.RemoveAll(deletePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove whiteout file: %w", err)
	}

	return nil
}

// extractTarEntry extracts a single tar entry to the target directory
func (f *LayerFlattener) extractTarEntry(targetDir string, header *tar.Header, reader io.Reader) error {
	// Sanitize path to prevent directory traversal
	targetPath := filepath.Join(targetDir, filepath.Clean(header.Name))

	// Make sure the path is still within targetDir
	if !strings.HasPrefix(targetPath, targetDir) {
		return fmt.Errorf("path traversal detected: %s", header.Name)
	}

	switch header.Typeflag {
	case tar.TypeDir:
		// Create directory
		if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		// Restore ownership if possible (may require root)
		_ = os.Lchown(targetPath, header.Uid, header.Gid)

	case tar.TypeReg:
		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("mkdir parent: %w", err)
		}

		file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer file.Close()

		if _, err := io.CopyN(file, reader, header.Size); err != nil && err != io.EOF {
			return fmt.Errorf("copy file content: %w", err)
		}

		// Restore ownership if possible (may require root)
		_ = os.Lchown(targetPath, header.Uid, header.Gid)

	case tar.TypeSymlink:
		// Create symlink (remove existing first)
		_ = os.Remove(targetPath)
		if err := os.Symlink(header.Linkname, targetPath); err != nil {
			return fmt.Errorf("create symlink: %w", err)
		}

	case tar.TypeLink:
		// Hard link - create a copy instead if target is outside rootfs
		linkTarget := filepath.Join(targetDir, filepath.Clean(header.Linkname))
		if !strings.HasPrefix(linkTarget, targetDir) {
			// Fallback: create empty file
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("mkdir parent: %w", err)
			}
			if _, err := os.Create(targetPath); err != nil {
				return fmt.Errorf("create hardlink fallback file: %w", err)
			}
		} else {
			// Create hard link
			if err := os.Link(linkTarget, targetPath); err != nil {
				return fmt.Errorf("create hardlink: %w", err)
			}
		}

	case tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
		// Skip special files (device nodes, pipes) - will be created by container on startup
		return nil

	default:
		// Unknown type - skip
		return nil
	}

	return nil
}
