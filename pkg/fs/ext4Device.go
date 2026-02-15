package fs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

type Ext4Builder struct{}

func NewExt4Builder() BlockDeviceBuilder {
	return &Ext4Builder{}
}

type Ext4Device struct {
	label     string
	sizeBytes int64
	path      string
}

func (d *Ext4Device) SizeBytes() int64 {
	return d.sizeBytes
}

func (d *Ext4Device) Label() string {
	return d.label
}

func (d *Ext4Device) Mount() (string, error) {
	mountDir := path.Join(os.TempDir(), d.mountDirName())
	if err := os.RemoveAll(mountDir); err != nil {
		return "", fmt.Errorf("removing ext4 mountdir: %w", err)
	}
	if err := os.Mkdir(mountDir, 0o755); err != nil {
		return "", fmt.Errorf("creating ext4 mountdir: %w", err)
	}

	out, err := exec.Command("sudo", "mount", d.path, mountDir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error mounting ext4 device to dir %s:\n%w\n%s", mountDir, err, out)
	}

	return mountDir, nil
}

func (d *Ext4Device) Unmount() error {
	mountDir := path.Join(os.TempDir(), d.mountDirName())
	// if mountDir does not exists nothing to unmount
	if _, err := os.Stat(mountDir); err != nil {
		return nil
	}

	if err := exec.Command("sudo", "umount", mountDir).Run(); err != nil {
		return fmt.Errorf("umounting ext4 device from %s : %w", mountDir, err)
	}

	if err := os.RemoveAll(mountDir); err != nil {
		return fmt.Errorf("removing mountdir %s: %w", mountDir, err)
	}

	return nil
}

func (d *Ext4Device) Path() string {
	return d.path
}

func (d *Ext4Device) mountDirName() string {
	fileName := path.Base(d.path)
	ext := path.Ext(fileName)

	return strings.ReplaceAll(fileName, ext, "") + "_mount"
}

// NewDevice heavily shells out for fs operations, maybe I ipmlement more in go later
func (b *Ext4Builder) NewDevice(ctx context.Context, opts BlockDeviceOptions) (BlockDevice, error) {
	// min save file size to write journal
	sizeBytes := max(opts.SizeBytes, int64(7*1024*1024))

	err := createSparseFile(opts.OutputFilePath, sizeBytes)
	if err != nil {
		return nil, fmt.Errorf("error createing sparse file: %w", err)
	}

	var out []byte
	if len(opts.Label) > 0 {
		out, err = exec.Command("mkfs.ext4", "-F", "-L", opts.Label, opts.OutputFilePath).CombinedOutput()
	} else {
		out, err = exec.Command("mkfs.ext4", "-F", opts.OutputFilePath).CombinedOutput()
	}
	if err != nil {
		return nil, fmt.Errorf("error formating file as ext4: %w \n%s", err, out)
	}

	return &Ext4Device{
		path:      opts.OutputFilePath,
		sizeBytes: opts.SizeBytes,
		label:     opts.Label,
	}, nil
}
