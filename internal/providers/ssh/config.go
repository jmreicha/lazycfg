package ssh

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmreicha/lazycfg/internal/core"
	"gopkg.in/yaml.v3"
)

//nolint:gochecknoinits // Required for registering provider config factory
func init() {
	core.RegisterProviderConfigFactory("ssh", func(raw map[string]interface{}) (core.ProviderConfig, error) {
		return ConfigFromMap(raw)
	})
}

// Config represents SSH provider-specific configuration.
type Config struct {
	// Enabled indicates whether this provider should be active
	Enabled bool `yaml:"enabled"`

	// ConfigPath is the path to the SSH configuration directory
	ConfigPath string `yaml:"config_path"`

	// Hosts contains SSH host configurations
	Hosts []HostConfig `yaml:"hosts"`
}

// HostConfig represents configuration for a single SSH host.
type HostConfig struct {
	// Host is the SSH host pattern
	Host string `yaml:"host"`

	// Hostname is the actual hostname or IP address
	Hostname string `yaml:"hostname"`

	// Port is the SSH port number
	Port int `yaml:"port"`

	// User is the SSH username
	User string `yaml:"user"`

	// IdentityFile is the path to the SSH private key
	IdentityFile string `yaml:"identity_file"`

	// Additional options
	Options map[string]string `yaml:"options"`
}

// ConfigFromMap builds a typed configuration from a raw provider map.
func ConfigFromMap(raw map[string]interface{}) (*Config, error) {
	cfg := DefaultConfig()
	if raw == nil {
		return cfg, nil
	}

	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ssh config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode ssh config: %w", err)
	}

	if cfg.ConfigPath == "" {
		cfg.ConfigPath = defaultConfigPath()
	}

	return cfg, nil
}

// DefaultConfig returns the default SSH provider configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:    true,
		ConfigPath: defaultConfigPath(),
		Hosts:      []HostConfig{},
	}
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}

	return filepath.Join(home, ".ssh")
}
