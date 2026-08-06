package az

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// lockFileMode is the mode applied to advisory lock files.
const lockFileMode = os.FileMode(0600)

// withExclusiveLock runs fn while holding an exclusive advisory lock on
// lockPath. A fresh *flock.Flock is constructed per acquisition so no lock
// state outlives the call, and the lock is always released. Acquisition blocks
// only as long as ctx allows; if ctx is cancelled first, its error is returned.
func withExclusiveLock(ctx context.Context, lockPath string, fn func() error) error {
	return withLock(ctx, lockPath, false, fn)
}

// withSharedLock runs fn while holding a shared advisory lock on lockPath,
// permitting concurrent readers. Semantics otherwise match withExclusiveLock.
func withSharedLock(ctx context.Context, lockPath string, fn func() error) error {
	return withLock(ctx, lockPath, true, fn)
}

func withLock(ctx context.Context, lockPath string, shared bool, fn func() error) error {
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// MkdirAll is subject to the process umask; force the mode.
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	// Pre-create with restrictive permissions; flock would otherwise create
	// the file subject to the process umask.
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, lockFileMode)
	if err != nil {
		return err
	}
	f.Close()
	if err = os.Chmod(lockPath, lockFileMode); err != nil {
		return err
	}

	l := flock.New(lockPath)
	if err = acquire(ctx, l, shared); err != nil {
		return err
	}
	defer l.Unlock()

	return fn()
}

// acquire blocks in flock's own blocking syscall on a helper goroutine so the
// wait is bounded only by ctx, with no polling interval or randomized backoff.
func acquire(ctx context.Context, l *flock.Flock, shared bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		if shared {
			done <- l.RLock()
			return
		}
		done <- l.Lock()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// Release whatever the helper eventually acquires so no lock leaks.
		go func() {
			if err := <-done; err == nil {
				l.Unlock()
			}
		}()
		return ctx.Err()
	}
}
