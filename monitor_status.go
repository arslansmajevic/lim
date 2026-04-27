package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type monitorStatus struct {
	PID           int       `json:"pid"`
	StartedAt     time.Time `json:"started_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	LastError     string    `json:"last_error,omitempty"`
}

func monitorStatusFilePath() (string, error) {
	stateFile, err := stateFilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(stateFile), "monitor.status.json"), nil
}

func writeMonitorStatus(st monitorStatus) error {
	path, err := monitorStatusFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create status dir: %w", err)
	}

	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	b = append(b, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write temp status: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace status: %w", err)
	}

	return nil
}

func readMonitorStatus() (monitorStatus, bool, error) {
	path, err := monitorStatusFilePath()
	if err != nil {
		return monitorStatus{}, false, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return monitorStatus{}, false, nil
		}
		return monitorStatus{}, false, fmt.Errorf("read status: %w", err)
	}
	var st monitorStatus
	if err := json.Unmarshal(b, &st); err != nil {
		return monitorStatus{}, false, fmt.Errorf("parse status: %w", err)
	}
	return st, true, nil
}

func clearMonitorStatus() error {
	path, err := monitorStatusFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
