package fs

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

func diskUsage(path string) (int64, error) {
	output, err := exec.Command("du", "-sb", path).Output()
	if err != nil {
		return 0, fmt.Errorf("error getting dir size: %w", err)
	}

	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected du output: %q", output)
	}

	sizeBytes, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse du size: %w", err)
	}

	return sizeBytes, nil
}

func createSparseFile(path string, sizeBytes int64) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer f.Close()

	_, err = f.Seek(sizeBytes-1, 0) // Seek to last byte position
	if err != nil {
		return fmt.Errorf("error geting last byte: %w", err)
	}

	// Write one byte (marks end of file, keeps rest sparse)
	_, err = f.Write([]byte{0})
	if err != nil {
		return fmt.Errorf("error writing last byte: %w", err)
	}
	// Result: sizeBytes allocation, but only ~4KB disk usage
	return nil
}

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
