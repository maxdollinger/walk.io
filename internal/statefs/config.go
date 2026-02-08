package statefs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StateFSConfig implements fs.BuilderConfig for state filesystems.
// It injects minimal metadata into an empty StateFS filesystem.
type StateFSConfig struct {
	AppID      string
	CreatedAt  time.Time
	Persistent bool // for future: ephemeral vs persistent
}

// WriteConfig injects minimal metadata into empty StateFS.
// This implements the fs.BuilderConfig interface.
func (c *StateFSConfig) WriteConfig(ctx context.Context, rootfsDir string) error {
	metaDir := filepath.Join(rootfsDir, "walkio")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return fmt.Errorf("create /walkio directory: %w", err)
	}

	// Write StateFS metadata
	metadataFile := filepath.Join(metaDir, "state")
	metadata := fmt.Sprintf("appid=%s\ncreated=%d\npersistent=%v\n",
		c.AppID, c.CreatedAt.Unix(), c.Persistent)

	if err := os.WriteFile(metadataFile, []byte(metadata), 0o644); err != nil {
		return fmt.Errorf("write state metadata: %w", err)
	}

	return nil
}
