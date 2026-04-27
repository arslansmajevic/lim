package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun_Help(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"lim", "help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected usage in stdout, got: %q", out.String())
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"lim", "nope"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("expected unknown command error, got: %q", errOut.String())
	}
}

func TestParseDockerEventsLine_ContainerCreate(t *testing.T) {
	line := "2026-04-27T07:29:50.123456789Z container create 0123456789ab (image=alpine:3.20, name=foo)"
	image, ts, ok := parseDockerEventsLine(line)
	if !ok {
		t.Fatalf("expected ok")
	}
	if image != "alpine:3.20" {
		t.Fatalf("unexpected image: %q", image)
	}
	if ts.IsZero() {
		t.Fatalf("expected non-zero timestamp")
	}
}

func TestListCmd_EmptyState(t *testing.T) {
	tmp := t.TempDir()
	stateFile := filepath.Join(tmp, "images.json")

	oldProvider := stateFilePathProvider
	stateFilePathProvider = func() (string, error) { return stateFile, nil }
	t.Cleanup(func() { stateFilePathProvider = oldProvider })

	var out, errOut bytes.Buffer
	code := run([]string{"lim", "list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", code, errOut.String())
	}
	if out.String() != "" {
		t.Fatalf("expected empty output, got %q", out.String())
	}
}

func TestListCmd_WithState(t *testing.T) {
	tmp := t.TempDir()
	stateFile := filepath.Join(tmp, "images.json")

	st, err := loadImageState(stateFile)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if err := st.setLastRun("alpine:3.20", time.Date(2026, 4, 27, 7, 29, 50, 0, time.UTC)); err != nil {
		t.Fatalf("set last run: %v", err)
	}

	oldProvider := stateFilePathProvider
	stateFilePathProvider = func() (string, error) { return stateFile, nil }
	t.Cleanup(func() { stateFilePathProvider = oldProvider })

	var out, errOut bytes.Buffer
	code := run([]string{"lim", "list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", code, errOut.String())
	}
	if !strings.HasPrefix(out.String(), "alpine:3.20\t") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestListCmd_BeforeFiltersOldImages(t *testing.T) {
	tmp := t.TempDir()
	stateFile := filepath.Join(tmp, "images.json")

	fixedNow := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	oldNowProvider := nowProvider
	nowProvider = func() time.Time { return fixedNow }
	t.Cleanup(func() { nowProvider = oldNowProvider })

	st, err := loadImageState(stateFile)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if err := st.setLastRun("old:1", fixedNow.Add(-25*time.Hour)); err != nil {
		t.Fatalf("set last run: %v", err)
	}
	if err := st.setLastRun("new:1", fixedNow.Add(-2*time.Hour)); err != nil {
		t.Fatalf("set last run: %v", err)
	}

	oldProvider := stateFilePathProvider
	stateFilePathProvider = func() (string, error) { return stateFile, nil }
	t.Cleanup(func() { stateFilePathProvider = oldProvider })

	var out, errOut bytes.Buffer
	code := run([]string{"lim", "list", "--before", "24h"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%q)", code, errOut.String())
	}
	if strings.Contains(out.String(), "new:1\t") {
		t.Fatalf("did not expect new image in output: %q", out.String())
	}
	if !strings.Contains(out.String(), "old:1\t") {
		t.Fatalf("expected old image in output: %q", out.String())
	}
}

func TestListCmd_BeforeRejectsNonHours(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"lim", "list", "--before", "2m"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}
