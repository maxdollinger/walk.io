package fs

import (
	"os"
	"path"
)

// WriteFileAtomic ensures atomic writes via rename. Beware that atomicity is only garantueed on the same filesystem
func WriteFileAtomic(filePath string, data []byte, perm os.FileMode) error {
	dir := path.Dir(filePath)
	tmp, err := os.CreateTemp(dir, "*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, filePath); err != nil {
		return err
	}

	// fsync dir so rename is durable across power loss
	dfd, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dfd.Close()
	return dfd.Sync()
}
