package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStateDir_PrefersSystemdUnitWhenPresent(t *testing.T) {
	oldGOOS := runtimeGOOS
	oldUnitPaths := systemdUnitPaths
	oldDefault := defaultSystemdStateDir
	t.Cleanup(func() {
		runtimeGOOS = oldGOOS
		systemdUnitPaths = oldUnitPaths
		defaultSystemdStateDir = oldDefault
	})

	runtimeGOOS = "linux"

	tmp := t.TempDir()
	unitPath := filepath.Join(tmp, "lim.service")
	if err := os.WriteFile(unitPath, []byte("[Service]\nEnvironment=LIM_STATE_DIR=/var/lib/lim-test\n"), 0o644); err != nil {
		t.Fatalf("write unit: %v", err)
	}
	systemdUnitPaths = []string{unitPath}
	defaultSystemdStateDir = "/var/lib/lim-default"

	if err := os.Unsetenv("LIM_STATE_DIR"); err != nil {
		t.Fatalf("unset env: %v", err)
	}

	dir, kind, err := resolveStateDir()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if kind != stateDirShared {
		t.Fatalf("expected shared kind, got %v", kind)
	}
	if dir != "/var/lib/lim-test" {
		t.Fatalf("expected systemd env dir, got %q", dir)
	}
}

func TestResolveStateDir_EnvOverridesSystemd(t *testing.T) {
	oldGOOS := runtimeGOOS
	oldUnitPaths := systemdUnitPaths
	t.Cleanup(func() {
		runtimeGOOS = oldGOOS
		systemdUnitPaths = oldUnitPaths
	})

	runtimeGOOS = "linux"

	tmp := t.TempDir()
	unitPath := filepath.Join(tmp, "lim.service")
	if err := os.WriteFile(unitPath, []byte("[Service]\nEnvironment=LIM_STATE_DIR=/var/lib/lim-test\n"), 0o644); err != nil {
		t.Fatalf("write unit: %v", err)
	}
	systemdUnitPaths = []string{unitPath}

	if err := os.Setenv("LIM_STATE_DIR", filepath.Join(tmp, "custom")); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("LIM_STATE_DIR") })

	dir, _, err := resolveStateDir()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if dir != filepath.Join(tmp, "custom") {
		t.Fatalf("expected env dir, got %q", dir)
	}
}
