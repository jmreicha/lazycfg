package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupManager_Backup(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.conf")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create backup
	backupPath, metadata, err := bm.Backup("testprovider", testFile)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	if backupPath == "" {
		t.Error("expected backup path, got empty string")
	}

	if metadata == nil {
		t.Fatal("expected metadata, got nil")
		return
	}

	if metadata.Provider != "testprovider" {
		t.Errorf("expected provider 'testprovider', got %q", metadata.Provider)
	}

	if metadata.OriginalPath != testFile {
		t.Errorf("expected original path %q, got %q", testFile, metadata.OriginalPath)
	}

	// Verify backup file exists and has correct content
	// #nosec G304 -- test file path from test temp directory
	content, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected content %q, got %q", testContent, string(content))
	}

	// Verify metadata file exists
	metadataPath := backupPath + ".json"
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("metadata file does not exist")
	}
}

func TestBackupManager_BackupNonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	_, _, err := bm.Backup("testprovider", "/nonexistent/file.conf")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestBackupManager_Restore(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.conf")
	originalContent := "original content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create backup
	backupPath, _, err := bm.Backup("testprovider", testFile)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Modify the original file
	modifiedContent := "modified content"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0o600); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Restore from backup
	if err := bm.Restore(backupPath); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify content was restored
	// #nosec G304 -- test file path from test temp directory
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(content) != originalContent {
		t.Errorf("expected content %q, got %q", originalContent, string(content))
	}
}

func TestBackupManager_List(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	// Create test files and backups with small delays to ensure unique timestamps
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("test%d.conf", i))
		if err := os.WriteFile(testFile, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if _, _, err := bm.Backup("testprovider", testFile); err != nil {
			t.Fatalf("Backup %d failed: %v", i, err)
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// List backups
	backups, err := bm.List("testprovider")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("expected 3 backups, got %d", len(backups))
	}
}

func TestBackupManager_ListNoBackups(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	backups, err := bm.List("nonexistent")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

func TestBackupManager_Clean(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	// Create test files and backups with delays to ensure unique timestamps
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tmpDir, fmt.Sprintf("test%d.conf", i))
		if err := os.WriteFile(testFile, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if _, _, err := bm.Backup("testprovider", testFile); err != nil {
			t.Fatalf("Backup %d failed: %v", i, err)
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Clean, keeping only 2 most recent
	if err := bm.Clean("testprovider", 2); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Verify only 2 backups remain
	backups, err := bm.List("testprovider")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("expected 2 backups after clean, got %d", len(backups))
	}
}
