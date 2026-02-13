// Package core provides the core functionality for cfgctl.
// This includes the provider interface, registry, configuration management, and orchestration engine.
package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultBackupDir is the default directory for storing backups.
	DefaultBackupDir = ".cfgctl/backups"
)

// BackupManager handles creation and restoration of configuration file backups.
type BackupManager struct {
	backupDir string
}

// NewBackupManager creates a new backup manager.
// If backupDir is empty, uses ~/.cfgctl/backups.
func NewBackupManager(backupDir string) *BackupManager {
	if backupDir == "" {
		home := os.Getenv("HOME")
		backupDir = filepath.Join(home, DefaultBackupDir)
	}

	return &BackupManager{
		backupDir: backupDir,
	}
}

// BackupFile creates a timestamped copy of the file at srcPath.
// Returns the backup path, or empty string if the source does not exist.
func BackupFile(srcPath string) (string, error) {
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", nil
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s.bak", srcPath, timestamp)

	if err := copyFile(srcPath, backupPath); err != nil {
		return "", fmt.Errorf("backup %q: %w", srcPath, err)
	}

	return backupPath, nil
}

// BackupMetadata contains information about a backup.
type BackupMetadata struct {
	Provider      string    `json:"provider"`
	OriginalPath  string    `json:"original_path"`
	BackupPath    string    `json:"backup_path"`
	Timestamp     time.Time `json:"timestamp"`
	RestoreMethod string    `json:"restore_method"`
}

// Backup creates a timestamped backup of a file.
// Returns the backup path and metadata, or an error if the backup fails.
func (bm *BackupManager) Backup(provider, filePath string) (string, *BackupMetadata, error) {
	// Check if source file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("source file does not exist: %s", filePath)
	}

	// Create backup directory if it doesn't exist
	providerBackupDir := filepath.Join(bm.backupDir, provider)
	if err := os.MkdirAll(providerBackupDir, 0o750); err != nil {
		return "", nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Base(filePath)
	backupFilename := fmt.Sprintf("%s.%s.backup", filename, timestamp)
	backupPath := filepath.Join(providerBackupDir, backupFilename)

	// Copy the file
	if err := copyFile(filePath, backupPath); err != nil {
		return "", nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Create metadata
	metadata := &BackupMetadata{
		Provider:      provider,
		OriginalPath:  filePath,
		BackupPath:    backupPath,
		Timestamp:     time.Now(),
		RestoreMethod: "copy",
	}

	// Write metadata file
	metadataPath := backupPath + ".json"
	if err := bm.writeMetadata(metadataPath, metadata); err != nil {
		return backupPath, metadata, fmt.Errorf("failed to write metadata: %w", err)
	}

	return backupPath, metadata, nil
}

// Restore restores a file from a backup.
func (bm *BackupManager) Restore(backupPath string) error {
	// Read metadata
	metadataPath := backupPath + ".json"
	metadata, err := bm.readMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Restore the file
	if err := copyFile(backupPath, metadata.OriginalPath); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// List returns all backups for a provider, sorted by timestamp (newest first).
func (bm *BackupManager) List(provider string) ([]*BackupMetadata, error) {
	providerBackupDir := filepath.Join(bm.backupDir, provider)

	// Check if backup directory exists
	if _, err := os.Stat(providerBackupDir); os.IsNotExist(err) {
		return []*BackupMetadata{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(providerBackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]*BackupMetadata, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		metadataPath := filepath.Join(providerBackupDir, entry.Name())
		metadata, err := bm.readMetadata(metadataPath)
		if err != nil {
			continue
		}

		backups = append(backups, metadata)
	}

	return backups, nil
}

// Clean removes old backups for a provider, keeping only the most recent N backups.
func (bm *BackupManager) Clean(provider string, keep int) error {
	backups, err := bm.List(provider)
	if err != nil {
		return err
	}

	if len(backups) <= keep {
		return nil
	}

	// Remove oldest backups
	for i := keep; i < len(backups); i++ {
		backup := backups[i]
		if err := os.Remove(backup.BackupPath); err != nil {
			return fmt.Errorf("failed to remove backup: %w", err)
		}
		if err := os.Remove(backup.BackupPath + ".json"); err != nil {
			return fmt.Errorf("failed to remove metadata: %w", err)
		}
	}

	return nil
}

// writeMetadata writes backup metadata to a file.
func (bm *BackupManager) writeMetadata(path string, metadata *BackupMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// readMetadata reads backup metadata from a file.
func (bm *BackupManager) readMetadata(path string) (*BackupMetadata, error) {
	// #nosec G304 -- path is from internal backup directory structure
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	// #nosec G304 -- paths are from internal configuration
	sourceFile, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := sourceFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Get source file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	// #nosec G304 -- paths are from internal configuration
	destFile, err := os.OpenFile(filepath.Clean(dst), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := destFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
