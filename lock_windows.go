//go:build windows

package main

type fileLock struct{}

func acquireLock(lockPath string) (*fileLock, bool, error) {
	// Not supported on Windows; allow multiple instances.
	return &fileLock{}, true, nil
}

func (l *fileLock) release() error { return nil }
