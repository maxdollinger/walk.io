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

func WriteContainerConfig(ctx context.Context, config *oci.ImageConfig, rootfsDir string) error {
	configDir := path.Join(rootfsDir, "walkio")
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		return fmt.Errorf("create walkio directory: %w", err)
	}

	err = writeAppEnv(configDir, config)
	if err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	err = writeAppArgv(configDir, config)
	if err != nil {
		return fmt.Errorf("write argv file: %w", err)
	}

	return nil
}

// writeEnv creates /walkio/env file with environment variables from image config.
func writeAppEnv(configDir string, config *oci.ImageConfig) error {
	var env bytes.Buffer
	writer := bufio.NewWriter(&env)

	for _, line := range config.Env {
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
	if len(config.WorkingDir) > 0 {
		workdir = config.WorkingDir
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
func writeAppArgv(configDir string, config *oci.ImageConfig) error {
	var argv bytes.Buffer
	writer := bufio.NewWriter(&argv)

	for _, line := range config.Entrypoint {
		_, err := writer.WriteString(strings.TrimSpace(line))
		if err != nil {
			return fmt.Errorf("write entrypoint to buffer: %w", err)
		}
		_, err = writer.WriteRune('\n')
		if err != nil {
			return fmt.Errorf("write newline to buffer: %w", err)
		}
	}

	for _, line := range config.Cmd {
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
