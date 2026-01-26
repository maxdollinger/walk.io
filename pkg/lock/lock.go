package lock

import (
	"context"

	"github.com/opencontainers/go-digest"
)

// Locker provides distributed locking for concurrent builds
// Blocks until lock is acquired or context is cancelled
type Locker interface {
	AcquireLock(ctx context.Context, digest digest.Digest) (Lock, error)
}

// Lock represents an acquired lock that must be released
type Lock interface {
	Release() error
}
