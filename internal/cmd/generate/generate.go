package generate

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jmreicha/lazycfg/internal/cmd/util"
	"github.com/lithammer/dedent"
)

var (
	// Default paths
	defaultHome                = os.Getenv("HOME")
	defaultAwsConfigPath       = filepath.Join(defaultHome, ".aws", "config")
	defaultGrantedConfigPath   = filepath.Join(defaultHome, ".granted", "config")
	defaultKubeConfigPath      = filepath.Join(defaultHome, ".kube", "config")
	defaultSteampipeConfigPath = filepath.Join(defaultHome, ".steampipe", "config", "aws.spc")

	// Paths, initialized with defaults, can be overridden by environment variables
	AwsConfigPath      = util.GetEnvOrDefault("AWS_CONFIG_PATH", defaultAwsConfigPath)
	GrantedConfigPath  = util.GetEnvOrDefault("GRANTED_CONFIG_PATH", defaultGrantedConfigPath)
	KubeConfigPath     = util.GetEnvOrDefault("KUBE_CONFIG_PATH", defaultKubeConfigPath)
	SteamipeConfigPath = util.GetEnvOrDefault("STEAMPIPE_CONFIG_PATH", defaultSteampipeConfigPath)
)

//go:embed templates/granted_config_darwin.tmpl
var grantedConfigTemplateDarwin string

//go:embed templates/granted_config_linux.tmpl
var grantedConfigTemplateLinux string

// getOSSpecificTemplate returns the appropriate template for the current operating system
func getOSTemplateGranted() string {
	switch runtime.GOOS {
	case "darwin":
		return grantedConfigTemplateDarwin
	case "linux":
		return grantedConfigTemplateLinux
	default:
		fmt.Println("Unknown operating system")
	}
	return ""
}

// CreateGrantedConfiguration creates a configuration file for Granted.
// It returns an error if the file operation fails.
func CreateGrantedConfiguration(config string) error {
	util.CheckCmd("granted")

	if err := util.BackupConfig(config); err != nil {
		return err
	}

	// Ensure the directory structure exists
	dir := filepath.Dir(config)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.Create(filepath.Clean(config))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Use the OS-specific template content
	configContent := strings.TrimSpace(getOSTemplateGranted())

	_, writeErr := file.WriteString(configContent)
	if writeErr != nil {
		return fmt.Errorf("failed to write to file: %w", writeErr)
	} else {
		println("Configuration file created successfully " + "'" + config + "'")
	}

	return nil
}

// CreateSteampipeConfiguration creates a configuration file for Steampipe.
// It returns an error if the file operation fails.
func CreateSteampipeConfiguration(config string) error {
	util.CheckCmd("steampipe")

	// Check if AwsConfigPath exists first, as it is required for Steampipe configuration
	if _, err := os.Stat(AwsConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("AWS configuration file not found at %q", AwsConfigPath)
	}

	if err := util.BackupConfig(config); err != nil {
		return err
	}

	// Ensure the directory structure exists
	dir := filepath.Dir(config)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.Create(filepath.Clean(config))
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
	} else {
		println("Configuration file created successfully " + "'" + config + "'")
	}

	return nil
}
