package util_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmreicha/lazycfg/internal/cmd/util"
)

func TestBackupConfig_NoFile(t *testing.T) {
	if err := util.BackupConfig(filepath.Join(t.TempDir(), "missing.conf")); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestBackupConfig_File(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	content := []byte("data")

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := util.BackupConfig(configPath); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatal("expected original file to be removed")
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}

	var backupPath string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "config.bak.") {
			backupPath = filepath.Join(tmpDir, name)
			break
		}
	}

	if backupPath == "" {
		t.Fatal("expected backup file")
	}

	backupData, err := os.ReadFile(backupPath) // #nosec G304 -- test file path from temp dir
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}

	if string(backupData) != string(content) {
		t.Fatalf("backup content mismatch: %s", string(backupData))
	}
}

func TestBackupConfig_Directory(t *testing.T) {
	if err := util.BackupConfig(t.TempDir()); err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	key := "LAZYCFG_ENV_TEST"
	defaultValue := "default"

	if value := util.GetEnvOrDefault(key, defaultValue); value != defaultValue {
		t.Fatalf("expected default value, got %s", value)
	}

	t.Setenv(key, "custom")
	if value := util.GetEnvOrDefault(key, defaultValue); value != "custom" {
		t.Fatalf("expected custom value, got %s", value)
	}
}
