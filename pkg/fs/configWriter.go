package fs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/maxdollinger/walk.io/pkg/oci"
)

type ConfigWriter interface {
	// Prepare injects /walk/argv and /walk/env into the rootfs
	WriteConfig(ctx context.Context, rootfsDir string, config *oci.ImageConfig) error
}

type AppConfigWriter struct{}

func NewAppConfigWriter() *AppConfigWriter {
	return &AppConfigWriter{}
}

func (i *AppConfigWriter) WriteConfig(ctx context.Context, rootfsDir string, config *oci.ImageConfig) error {
	configDir := path.Join(rootfsDir, "walkio")
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		return fmt.Errorf("could not create config dir: %w", err)
	}

	err = i.writeEnv(configDir, config)
	if err != nil {
		return fmt.Errorf("could not create env file: %w", err)
	}

	err = i.writeArgv(configDir, config)
	if err != nil {
		return fmt.Errorf("could not create argv file: %w", err)
	}

	return nil
}

func (i *AppConfigWriter) writeEnv(configDir string, config *oci.ImageConfig) error {
	var env bytes.Buffer
	writer := bufio.NewWriter(&env)

	for _, line := range config.Env {
		_, err := writer.WriteString(strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("could not write env to buffer: %w", err)
		}
		_, err = writer.WriteRune('\n')
		if err != nil {
			return fmt.Errorf("could not write env to buffer: %w", err)
		}
	}

	workdir := "/"
	if len(config.WorkingDir) > 0 {
		workdir = config.WorkingDir
	}
	_, err := fmt.Fprintf(writer, "WORKDIR=%s", workdir)
	if err != nil {
		return fmt.Errorf("could not write workdir to buffer: %w", err)
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("could not flush env writer: %w", err)
	}

	envFilePath := path.Join(configDir, "env")
	err = os.WriteFile(envFilePath, env.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("could not write env to file: %w", err)
	}

	return nil
}

func (i *AppConfigWriter) writeArgv(configDir string, config *oci.ImageConfig) error {
	var argv bytes.Buffer
	writer := bufio.NewWriter(&argv)

	for _, line := range config.Entrypoint {
		_, err := writer.WriteString(strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("could not write arg to buffer: %w", err)
		}
		_, err = writer.WriteRune('\n')
		if err != nil {
			return fmt.Errorf("could not write arg to buffer: %w", err)
		}
	}

	for _, line := range config.Cmd {
		_, err := writer.WriteString(strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("could not write arg to buffer: %w", err)
		}
		_, err = writer.WriteRune('\n')
		if err != nil {
			return fmt.Errorf("could not write arg to buffer: %w", err)
		}
	}

	err := writer.Flush()
	if err != nil {
		return fmt.Errorf("could not flush argv writer: %w", err)
	}

	argvFilePath := path.Join(configDir, "argv")
	err = os.WriteFile(argvFilePath, argv.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("could not write argv to file: %w", err)
	}

	return nil
}

type NoOpFilesystemPreparer struct{}

func NewNoOpFilesystemPreparer() *NoOpFilesystemPreparer {
	return &NoOpFilesystemPreparer{}
}

func (p *NoOpFilesystemPreparer) WriteConfig(ctx context.Context, rootfsDir string, config *oci.ImageConfig) error {
	// No-op: in real implementation, would create /walk/argv and /walk/env
	return nil
}
