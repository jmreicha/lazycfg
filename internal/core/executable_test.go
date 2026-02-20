package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExecutable_ExecutableFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "mybin")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if !isExecutable(path) {
		t.Error("expected isExecutable to return true for executable file")
	}
}

func TestIsExecutable_NonExecutableFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "noexec")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if isExecutable(path) {
		t.Error("expected isExecutable to return false for non-executable file")
	}
}

func TestIsExecutable_NonexistentFile(t *testing.T) {
	if isExecutable("/nonexistent/path/to/file") {
		t.Error("expected isExecutable to return false for nonexistent file")
	}
}

func TestIsExecutable_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	if isExecutable(tmpDir) {
		t.Error("expected isExecutable to return false for directory")
	}
}

func TestCommonExecutablePaths_NonEmpty(t *testing.T) {
	paths := commonExecutablePaths()
	if len(paths) == 0 {
		t.Fatal("expected non-empty list of executable paths")
	}

	for _, p := range paths {
		if p == "" {
			t.Error("expected all paths to be non-empty strings")
		}
	}
}

func TestFindExecutable_WithHook_Found(t *testing.T) {
	findExecutableHook = func(name string) (string, bool) {
		if name == "testbin" {
			return "/usr/local/bin/testbin", true
		}
		return "", false
	}
	t.Cleanup(func() {
		findExecutableHook = nil
	})

	path, found := FindExecutable("testbin")
	if !found {
		t.Fatal("expected FindExecutable to return found=true")
	}
	if path != "/usr/local/bin/testbin" {
		t.Errorf("path = %q, want /usr/local/bin/testbin", path)
	}
}

func TestFindExecutable_WithHook_NotFound(t *testing.T) {
	findExecutableHook = func(string) (string, bool) {
		return "", false
	}
	t.Cleanup(func() {
		findExecutableHook = nil
	})

	path, found := FindExecutable("nonexistent")
	if found {
		t.Fatal("expected FindExecutable to return found=false")
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

func TestFindExecutable_RealBinary(t *testing.T) {
	// "ls" should always exist on darwin/linux
	findExecutableHook = nil
	path, found := FindExecutable("ls")
	if !found {
		t.Skip("ls not found on this system, skipping")
	}
	if path == "" {
		t.Error("expected non-empty path for ls")
	}
}

func TestFindExecutable_NotExistReal(t *testing.T) {
	findExecutableHook = nil
	_, found := FindExecutable("cfgctl_definitely_not_a_real_binary_xyz")
	if found {
		t.Error("expected FindExecutable to return false for nonexistent binary")
	}
}

func TestMissingExecutables_AllPresent(t *testing.T) {
	findExecutableHook = func(string) (string, bool) {
		return "/bin/found", true
	}
	t.Cleanup(func() {
		findExecutableHook = nil
	})

	missing := MissingExecutables("a", "b", "c")
	if len(missing) != 0 {
		t.Errorf("expected no missing executables, got %v", missing)
	}
}

func TestMissingExecutables_SomeMissing(t *testing.T) {
	findExecutableHook = func(name string) (string, bool) {
		if name == "b" {
			return "", false
		}
		return "/bin/" + name, true
	}
	t.Cleanup(func() {
		findExecutableHook = nil
	})

	missing := MissingExecutables("a", "b", "c")
	if len(missing) != 1 || missing[0] != "b" {
		t.Errorf("expected [b], got %v", missing)
	}
}

func TestMissingExecutables_Empty(t *testing.T) {
	missing := MissingExecutables()
	if len(missing) != 0 {
		t.Errorf("expected empty slice, got %v", missing)
	}
}
