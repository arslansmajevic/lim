package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type imageState struct {
	filePath string
	images   map[string]time.Time
}

var stateFilePathProvider = defaultStateFilePath

// These vars exist so tests can override filesystem locations and OS detection.
var runtimeGOOS = runtime.GOOS

var systemdUnitPaths = []string{
	"/etc/systemd/system/lim.service",
	"/lib/systemd/system/lim.service",
	"/usr/lib/systemd/system/lim.service",
}

var defaultSystemdStateDir = "/var/lib/lim"

type stateDirKind int

const (
	stateDirUser stateDirKind = iota
	stateDirShared
)

func stateFilePath() (string, error) {
	return stateFilePathProvider()
}

func defaultStateFilePath() (string, error) {
	stateDir, _, err := resolveStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "images.json"), nil
}

func resolveStateDir() (dir string, kind stateDirKind, err error) {
	if d := os.Getenv("LIM_STATE_DIR"); d != "" {
		return filepath.Clean(d), stateDirShared, nil
	}

	if d, ok := detectSystemdStateDir(); ok {
		return d, stateDirShared, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", stateDirUser, fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(configDir, "lim"), stateDirUser, nil
}

func detectSystemdStateDir() (string, bool) {
	if runtimeGOOS != "linux" {
		return "", false
	}

	unitPath := ""
	for _, p := range systemdUnitPaths {
		if _, err := os.Stat(p); err == nil {
			unitPath = p
			break
		}
	}
	if unitPath == "" {
		return "", false
	}

	// If the unit file defines LIM_STATE_DIR, use it.
	if d, ok := parseSystemdUnitEnv(unitPath, "LIM_STATE_DIR"); ok {
		return filepath.Clean(d), true
	}

	// Fallback to the default used by our installer.
	return filepath.Clean(defaultSystemdStateDir), true
}

func parseSystemdUnitEnv(unitPath string, key string) (string, bool) {
	f, err := os.Open(unitPath)
	if err != nil {
		return "", false
	}
	defer f.Close()

	needle := key + "="
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "Environment=") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "Environment="))
		rest = strings.Trim(rest, "\"")
		idx := strings.Index(rest, needle)
		if idx < 0 {
			continue
		}
		val := rest[idx+len(needle):]
		// Value ends at whitespace or quote.
		val = strings.TrimLeft(val, "\"")
		for i, r := range val {
			if r == ' ' || r == '\t' || r == '"' {
				val = val[:i]
				break
			}
		}
		val = strings.TrimSpace(val)
		if val == "" {
			return "", false
		}
		return val, true
	}
	return "", false
}

func stateDirMode() os.FileMode {
	_, kind, err := resolveStateDir()
	if err == nil && kind == stateDirShared {
		return 0o755
	}
	return 0o700
}

func stateFileMode() os.FileMode {
	_, kind, err := resolveStateDir()
	if err == nil && kind == stateDirShared {
		return 0o644
	}
	return 0o600
}

func lockFileMode() os.FileMode {
	_, kind, err := resolveStateDir()
	if err == nil && kind == stateDirShared {
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
