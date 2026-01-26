package lock

import (
	"context"

	"github.com/opencontainers/go-digest"
)

type NoOpLocker struct{}

func NewNoOpLocker() *NoOpLocker {
	return &NoOpLocker{}
}

func (l *NoOpLocker) AcquireLock(ctx context.Context, digest digest.Digest) (Lock, error) {
	return &noopLock{}, nil
}

type noopLock struct{}

func (l *noopLock) Release() error {
	return nil
}
