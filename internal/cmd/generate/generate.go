package generate

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lithammer/dedent"
)

var (
	Home              = os.Getenv("HOME")
	GrantedConfigPath = Home + "/.granted/config"
)

// CreateToolConfiguration creates a configuration file for the tool.
// It returns an error if the file operation fails.
func CreateGrantedConfiguration() error {
	cmd := "granted"

	if _, err := exec.LookPath(cmd); err != nil {
		fmt.Printf("Command '%s' not found\n", cmd)
		fmt.Println("Please install and try again")
		os.Exit(1)
	}

	file, err := os.Create(GrantedConfigPath)
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
