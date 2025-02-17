package generate

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lithammer/dedent"
)

var (
	Home               = os.Getenv("HOME")
	GrantedConfigPath  = Home + "/.granted/config"
	SteamipeConfigPath = Home + "/.steampipe/config/aws.spc"
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
	fmt.Println("Existing configuration found, backing up to" + "'" + backupFilePath + "'")

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

// CreateGrantedConfiguration creates a configuration file for Granted.
// It returns an error if the file operation fails.
func CreateGrantedConfiguration() error {
	CheckCmd("granted")

	if err := BackupConfig(GrantedConfigPath); err != nil {
		return err
	}

	// Ensure the directory structure exists
	dir := filepath.Dir(GrantedConfigPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.Create(filepath.Clean(GrantedConfigPath))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	configContent := dedent.Dedent(`
		DefaultBrowser = "STDOUT"
		CustomBrowserPath = ""
		CustomSSOBrowserPath = ""
		Ordering = ""
		ExportCredentialSuffix = ""
		DisableUsageTips = true
		CredentialProcessAutoLogin = true
	`)
	configContent = strings.TrimSpace(configContent)

	_, writeErr := file.WriteString(configContent)
	if writeErr != nil {
		return fmt.Errorf("failed to write to file: %w", writeErr)
	}

	return nil
}

// CreateSteampipeConfiguration creates a configuration file for Steampipe.
// It returns an error if the file operation fails.
func CreateSteampipeConfiguration() error {
	CheckCmd("steampipe")

	if err := BackupConfig(SteamipeConfigPath); err != nil {
		return err
	}

	// Ensure the directory structure exists
	dir := filepath.Dir(SteamipeConfigPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.Create(filepath.Clean(SteamipeConfigPath))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	configContent := dedent.Dedent(`
		# This config file is auto-generated by lazycfg, do not edit

		connection "aws_all" {
			plugin      = "aws"
			type        = "aggregator"
			connections = ["aws_*"]
		}
	`)
	configContent = strings.TrimSpace(configContent)

	_, writeErr := file.WriteString(configContent)
	if writeErr != nil {
		return fmt.Errorf("failed to write to file: %w", writeErr)
	}

	return nil
}
