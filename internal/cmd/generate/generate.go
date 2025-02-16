package generate

import (
	"fmt"
	"os"
	"os/exec"
)

// CreateToolConfiguration creates a configuration file for the tool.
// It returns an error if the file operation fails.
func CreateGrantedConfiguration() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	filePath := homeDir + "/.granted/config"
	cmd := "granted"

	if _, err := exec.LookPath(cmd); err != nil {
		fmt.Printf("Command '%s' not found\n", cmd)
		fmt.Println("Please install and try again")
		os.Exit(1)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	configContent := `DefaultBrowser = "STDOUT"
CustomBrowserPath = ""
CustomSSOBrowserPath = ""
Ordering = ""
ExportCredentialSuffix = ""
DisableUsageTips = true
CredentialProcessAutoLogin = true`

	_, writeErr := file.WriteString(configContent)
	if writeErr != nil {
		return fmt.Errorf("failed to write to file: %w", writeErr)
	}

	return nil
}
