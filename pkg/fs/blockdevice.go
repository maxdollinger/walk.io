package fs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type BlockDeviceBuilder interface {
	// Creates an ext4 image with minimal size for the content
	NewDevice(ctx context.Context, opts BlockDeviceOptions) (*BlockDevice, error)
}

// BlockDeviceOptions specifies how to create a block device
type BlockDeviceOptions struct {
	BuildDirPath   string // build dir to craete mount point
	SourceDirPath  string // prepared rootfs directory
	OutputFilePath string // Path of the file to be written including the file itself
	Label          string // filesystem label (optional)
}

type BlockDevice struct {
	Path      string
	SizeBytes int64
	Label     string
}

type Ext4Builder struct{}

func NewExt4Builder() BlockDeviceBuilder {
	return &Ext4Builder{}
}

// NewDevice heavily shells out for fs operations, maybe I ipmlement more in go later
func (b *Ext4Builder) NewDevice(ctx context.Context, opts BlockDeviceOptions) (*BlockDevice, error) {
	sizeBytes, err := diskUsage(opts.SourceDirPath)
	if err != nil {
		return nil, fmt.Errorf("error calculating source folder size: %w", err)
	}
	// include a 15% size buffer
	actualSize := sizeBytes * 115 / 100

	err = createSparseFile(opts.OutputFilePath, actualSize)
	if err != nil {
		return nil, fmt.Errorf("error createing sparse file: %w", err)
	}

	label := "APP"
	if len(opts.Label) > 0 {
		label = opts.Label
	}
	err = exec.Command("mkfs.ext4", "-F", "-L", label, opts.OutputFilePath).Run()
	if err != nil {
		return nil, fmt.Errorf("error formating file as ext4: %w", err)
	}

	mntDir := filepath.Join(opts.BuildDirPath, "mnt")
	if err := os.MkdirAll(mntDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	out, err := exec.Command("mount", "-t", "ext4", "-o", "loop", opts.OutputFilePath, mntDir).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error mounting app device: %w\n%s", err, string(out))
	}
	defer func() {
		_ = exec.Command("umount", mntDir).Run()
	}()

	// copy files
	err = exec.Command("cp", "-a", "--", fmt.Sprintf("%s/.", opts.SourceDirPath), fmt.Sprintf("%s/.", mntDir)).Run()
	if err != nil {
		return nil, fmt.Errorf("faild copying files to device: %w", err)
	}

	return &BlockDevice{
		Path:      opts.OutputFilePath,
		SizeBytes: actualSize,
		Label:     label,
	}, nil
}

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
