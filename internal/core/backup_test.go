package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	for i := range 3 {
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
	for i := range 5 {
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

func TestBackupFile_NoFile(t *testing.T) {
	backup, err := BackupFile(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}
	if backup != "" {
		t.Fatalf("expected empty backup path, got %q", backup)
	}
}

func TestBackupManager_Restore_MissingMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	// Try to restore from a path whose .json metadata file does not exist
	err := bm.Restore(filepath.Join(tmpDir, "nonexistent.backup"))
	if err == nil {
		t.Fatal("expected error for missing metadata, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read metadata") {
		t.Errorf("expected 'failed to read metadata' in error, got %q", err.Error())
	}
}

func TestBackupManager_Restore_MissingBackupFile(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	// Create a test file, back it up, then delete the backup file (but keep metadata)
	testFile := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(testFile, []byte("content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backupPath, _, err := bm.Backup("testprovider", testFile)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Remove the backup file but keep the metadata .json
	if err := os.Remove(backupPath); err != nil {
		t.Fatalf("failed to remove backup file: %v", err)
	}

	err = bm.Restore(backupPath)
	if err == nil {
		t.Fatal("expected error for missing backup file, got nil")
	}
	if !strings.Contains(err.Error(), "backup file does not exist") {
		t.Errorf("expected 'backup file does not exist' in error, got %q", err.Error())
	}
}

func TestBackupManager_Restore_InvalidMetadataJSON(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	backupPath := filepath.Join(tmpDir, "fake.backup")
	metadataPath := backupPath + ".json"

	// Write a backup file and invalid JSON metadata
	if err := os.WriteFile(backupPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to write backup file: %v", err)
	}
	if err := os.WriteFile(metadataPath, []byte("not json{{{"), 0o600); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	err := bm.Restore(backupPath)
	if err == nil {
		t.Fatal("expected error for invalid metadata JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read metadata") {
		t.Errorf("expected 'failed to read metadata' in error, got %q", err.Error())
	}
}

func TestBackupManager_Clean_NothingToClean(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	// Clean with no backups should succeed
	if err := bm.Clean("nonexistent", 5); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}
}

func TestBackupManager_Clean_KeepMoreThanExist(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(filepath.Join(tmpDir, "backups"))

	testFile := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(testFile, []byte("content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if _, _, err := bm.Backup("testprovider", testFile); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Keep 10 but only 1 exists -- should be a no-op
	if err := bm.Clean("testprovider", 10); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	backups, err := bm.List("testprovider")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("expected 1 backup, got %d", len(backups))
	}
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyFile(filepath.Join(tmpDir, "missing"), filepath.Join(tmpDir, "dst"))
	if err == nil {
		t.Fatal("expected error for missing source file, got nil")
	}
}

func TestCopyFile_DestDirNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	if err := os.WriteFile(src, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	err := copyFile(src, filepath.Join(tmpDir, "nodir", "dst"))
	if err == nil {
		t.Fatal("expected error for nonexistent dest directory, got nil")
	}
}

func TestWriteMetadata_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(tmpDir)

	metadata := &BackupMetadata{
		Provider:      "test",
		OriginalPath:  "/tmp/original",
		BackupPath:    "/tmp/backup",
		Timestamp:     time.Now().Truncate(time.Second),
		RestoreMethod: "copy",
	}

	path := filepath.Join(tmpDir, "meta.json")
	if err := bm.writeMetadata(path, metadata); err != nil {
		t.Fatalf("writeMetadata failed: %v", err)
	}

	got, err := bm.readMetadata(path)
	if err != nil {
		t.Fatalf("readMetadata failed: %v", err)
	}

	if got.Provider != metadata.Provider {
		t.Errorf("Provider = %q, want %q", got.Provider, metadata.Provider)
	}
	if got.OriginalPath != metadata.OriginalPath {
		t.Errorf("OriginalPath = %q, want %q", got.OriginalPath, metadata.OriginalPath)
	}
	if got.RestoreMethod != metadata.RestoreMethod {
		t.Errorf("RestoreMethod = %q, want %q", got.RestoreMethod, metadata.RestoreMethod)
	}
}

func TestReadMetadata_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(tmpDir)

	_, err := bm.readMetadata(filepath.Join(tmpDir, "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing metadata file, got nil")
	}
}

func TestReadMetadata_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	bm := NewBackupManager(tmpDir)

	path := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := bm.readMetadata(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestBackupFile_CreatesCopy(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "config")
	content := []byte("config data")

	if err := os.WriteFile(srcPath, content, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	backup, err := BackupFile(srcPath)
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}
	if !strings.HasPrefix(backup, srcPath+".") || !strings.HasSuffix(backup, ".bak") {
		t.Fatalf("backup = %q, expected timestamped .bak file", backup)
	}

	data, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("backup content = %q", string(data))
	}
}
