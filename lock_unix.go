//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type fileLock struct {
	f *os.File
}

// acquireLock tries to take an exclusive lock on lockPath.
// Returns (lock, acquired, err). If acquired is false and err is nil, another instance holds the lock.
func acquireLock(lockPath string) (*fileLock, bool, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return nil, false, fmt.Errorf("create lock dir: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("lock file: %w", err)
	}

	return &fileLock{f: f}, true, nil
}

func (l *fileLock) release() error {
	if l == nil || l.f == nil {
		return nil
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	return l.f.Close()
}

func (l *fileLock) setPID(pid int) error {
	if l == nil || l.f == nil {
		return nil
	}
	if err := l.f.Truncate(0); err != nil {
		return err
	}
	if _, err := l.f.Seek(0, 0); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(l.f, "%d\n", pid); err != nil {
		return err
	}
	return l.f.Sync()
}
