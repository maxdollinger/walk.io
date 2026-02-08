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

// AppConfigWriter implements BuilderConfig for application filesystems.
// It injects OCI image metadata (/walkio/env, /walkio/argv) into the rootfs.
type AppConfigWriter struct {
	imageConfig *oci.ImageConfig
}

// NewAppConfigWriter creates a new AppConfigWriter for the given image config.
func NewAppConfigWriter(imageConfig *oci.ImageConfig) *AppConfigWriter {
	return &AppConfigWriter{
		imageConfig: imageConfig,
	}
}

// WriteConfig injects /walkio/env and /walkio/argv into the rootfs.
// This implements the BuilderConfig interface.
func (w *AppConfigWriter) WriteConfig(ctx context.Context, rootfsDir string) error {
	configDir := path.Join(rootfsDir, "walkio")
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		return fmt.Errorf("create walkio directory: %w", err)
	}

	err = w.writeEnv(configDir)
	if err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	err = w.writeArgv(configDir)
	if err != nil {
		return fmt.Errorf("write argv file: %w", err)
	}

	return nil
}

// writeEnv creates /walkio/env file with environment variables from image config.
func (w *AppConfigWriter) writeEnv(configDir string) error {
	var env bytes.Buffer
	writer := bufio.NewWriter(&env)

	for _, line := range w.imageConfig.Env {
		_, err := writer.WriteString(strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("write env to buffer: %w", err)
		}
		_, err = writer.WriteRune('\n')
		if err != nil {
			return fmt.Errorf("write newline to buffer: %w", err)
		}
	}

	workdir := "/"
	if len(w.imageConfig.WorkingDir) > 0 {
		workdir = w.imageConfig.WorkingDir
	}
	_, err := fmt.Fprintf(writer, "WORKDIR=%s", workdir)
	if err != nil {
		return fmt.Errorf("write workdir to buffer: %w", err)
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("flush env writer: %w", err)
	}

	envFilePath := path.Join(configDir, "env")
	err = os.WriteFile(envFilePath, env.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	return nil
}

// writeArgv creates /walkio/argv file with entrypoint and cmd from image config.
func (w *AppConfigWriter) writeArgv(configDir string) error {
	var argv bytes.Buffer
	writer := bufio.NewWriter(&argv)

	for _, line := range w.imageConfig.Entrypoint {
		_, err := writer.WriteString(strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("write entrypoint to buffer: %w", err)
		}
		_, err = writer.WriteRune('\n')
		if err != nil {
			return fmt.Errorf("write newline to buffer: %w", err)
		}
	}

	for _, line := range w.imageConfig.Cmd {
		_, err := writer.WriteString(strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("write cmd to buffer: %w", err)
		}
		_, err = writer.WriteRune('\n')
		if err != nil {
			return fmt.Errorf("write newline to buffer: %w", err)
		}
	}

	err := writer.Flush()
	if err != nil {
		return fmt.Errorf("flush argv writer: %w", err)
	}

	argvFilePath := path.Join(configDir, "argv")
	err = os.WriteFile(argvFilePath, argv.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("write argv file: %w", err)
	}

	return nil
}

// NoOpBuilderConfig implements BuilderConfig with no-op behavior.
// Useful for testing or when no config injection is needed.
type NoOpBuilderConfig struct{}

func NewNoOpBuilderConfig() *NoOpBuilderConfig {
	return &NoOpBuilderConfig{}
}

func (p *NoOpBuilderConfig) WriteConfig(ctx context.Context, rootfsDir string) error {
	// No-op: do nothing
	return nil
}
