package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStateDir_DefaultIsUserConfigDir(t *testing.T) {
	if err := os.Unsetenv("LIM_STATE_DIR"); err != nil {
		t.Fatalf("unset LIM_STATE_DIR: %v", err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("user config dir: %v", err)
	}
	expected := filepath.Join(configDir, "lim")

	dir, kind, err := resolveStateDir()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if kind != stateDirUser {
		t.Fatalf("expected user kind, got %v", kind)
	}
	if dir != expected {
		t.Fatalf("expected user config dir, got %q", dir)
	}
}

func TestResolveStateDir_EnvOverridesDefault(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Setenv("LIM_STATE_DIR", filepath.Join(tmp, "custom")); err != nil {
		t.Fatalf("set LIM_STATE_DIR: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("LIM_STATE_DIR") })

	dir, kind, err := resolveStateDir()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if kind != stateDirShared {
		t.Fatalf("expected shared kind, got %v", kind)
	}
	if dir != filepath.Join(tmp, "custom") {
		t.Fatalf("expected env dir, got %q", dir)
	}
}
