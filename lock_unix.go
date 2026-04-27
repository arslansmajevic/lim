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

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, lockFileMode())
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

// isLockHeld reports whether another process currently holds the lock at lockPath.
// It does not create the file, and returns (false, nil) when the file doesn't exist.
func isLockHeld(lockPath string) (bool, error) {
	f, err := os.OpenFile(lockPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			return true, nil
		}
		return false, err
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false, nil
}
