package util

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// CheckCmd checks if a command is available in the system.
func CheckCmd(cmd string) {
	if _, err := exec.LookPath(cmd); err != nil {
		fmt.Printf("Command '%s' not found\n", cmd)
		fmt.Println("Please install and try again")
		os.Exit(1)
	}
}

// BackupConfig copies the given file to a new file with a `.bak` extension
// and a timestamp appended. It returns an error if the backup operation fails.
func BackupConfig(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		fmt.Printf("File %q does not exist, creating.\n", filePath)
		return nil
	} else if err != nil {
		return fmt.Errorf("unable to access file %q: %w", filePath, err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%q is a directory, not a file", filePath)
	}

	timestamp := time.Now().Format("200601021504")
	backupFilePath := filePath + ".bak." + timestamp
	fmt.Println("Existing configuration found, backing up to " + "'" + backupFilePath + "'")

	srcFile, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(filepath.Clean(backupFilePath))
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Remove the original file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	return nil
}
