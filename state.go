package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type imageState struct {
	filePath string
	images   map[string]time.Time
}

var stateFilePathProvider = defaultStateFilePath

func stateFilePath() (string, error) {
	return stateFilePathProvider()
}

func defaultStateFilePath() (string, error) {
	if dir := os.Getenv("LIM_STATE_DIR"); dir != "" {
		stateDir := filepath.Clean(dir)
		return filepath.Join(stateDir, "images.json"), nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	stateDir := filepath.Join(configDir, "lim")
	return filepath.Join(stateDir, "images.json"), nil
}

func stateDirMode() os.FileMode {
	if os.Getenv("LIM_STATE_DIR") != "" {
		return 0o755
	}
	return 0o700
}

func stateFileMode() os.FileMode {
	if os.Getenv("LIM_STATE_DIR") != "" {
		return 0o644
	}
	return 0o600
}

func lockFileMode() os.FileMode {
	if os.Getenv("LIM_STATE_DIR") != "" {
		// Allow any user to open the file (acquireLock uses O_RDWR).
		return 0o666
	}
	return 0o600
}

func loadImageState(filePath string) (*imageState, error) {
	st := &imageState{filePath: filePath, images: map[string]time.Time{}}

	b, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return st, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}

	var disk map[string]string
	if err := json.Unmarshal(b, &disk); err != nil {
		return nil, fmt.Errorf("parse state json: %w", err)
	}

	for image, tsStr := range disk {
		ts, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			// Skip invalid timestamps rather than failing entirely.
			continue
		}
		st.images[image] = ts
	}

	return st, nil
}

func (s *imageState) setLastRun(image string, ts time.Time) error {
	if image == "" {
		return nil
	}
	if s.images == nil {
		s.images = map[string]time.Time{}
	}

	prev, ok := s.images[image]
	if ok && !ts.After(prev) {
		return nil
	}

	s.images[image] = ts
	return s.save()
}

func (s *imageState) save() error {
	stateDir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(stateDir, stateDirMode()); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	// Write deterministically (sorted keys) for easier diffing and testing.
	keys := make([]string, 0, len(s.images))
	for k := range s.images {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	disk := make(map[string]string, len(s.images))
	for _, k := range keys {
		disk[k] = s.images[k].UTC().Format(time.RFC3339Nano)
	}

	b, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state json: %w", err)
	}
	b = append(b, '\n')

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, b, stateFileMode()); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, s.filePath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace state: %w", err)
	}

	return nil
}
