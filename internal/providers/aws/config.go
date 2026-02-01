package aws

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmreicha/lazycfg/internal/core"
	"gopkg.in/yaml.v3"
)

//nolint:gochecknoinits // Required for registering provider config factory
func init() {
	core.RegisterProviderConfigFactory("aws", func(raw map[string]interface{}) (core.ProviderConfig, error) {
		return ConfigFromMap(raw)
	})
}

const defaultSSOSessionName = "lazycfg"

var (
	errSSORegionEmpty       = errors.New("sso region cannot be empty")
	errSSOStartURLEmpty     = errors.New("sso start url cannot be empty")
	errTokenCachePathsEmpty = errors.New("token cache paths cannot be empty")
)

// Config represents AWS provider-specific configuration.
type Config struct {
	// Enabled indicates whether this provider should be active.
	Enabled bool `yaml:"enabled"`

	// Roles limits discovery to matching role names.
	Roles []string `yaml:"roles"`

	// SSO contains shared SSO configuration.
	SSO SSOConfig `yaml:"sso"`

	// TokenCachePaths lists cache locations for SSO tokens.
	TokenCachePaths []string `yaml:"token_cache_paths"`
}

// SSOConfig represents shared SSO configuration.
type SSOConfig struct {
	Region      string `yaml:"region"`
	SessionName string `yaml:"session_name"`
	StartURL    string `yaml:"start_url"`
}

// ConfigFromMap builds a typed configuration from a raw provider map.
func ConfigFromMap(raw map[string]interface{}) (*Config, error) {
	cfg := DefaultConfig()
	if raw == nil {
		return cfg, nil
	}

	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to encode aws config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode aws config: %w", err)
	}

	if cfg.TokenCachePaths == nil {
		cfg.TokenCachePaths = defaultTokenCachePaths()
	}

	if cfg.SSO.SessionName == "" {
		cfg.SSO.SessionName = defaultSSOSessionName
	}

	return cfg, nil
}

// DefaultConfig returns the default AWS provider configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:         true,
		Roles:           []string{},
		SSO:             defaultSSOConfig(),
		TokenCachePaths: defaultTokenCachePaths(),
	}
}

// Validate checks if the provider configuration is valid.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("aws config is nil")
	}

	if strings.TrimSpace(c.SSO.StartURL) == "" {
		return errSSOStartURLEmpty
	}

	if strings.TrimSpace(c.SSO.Region) == "" {
		return errSSORegionEmpty
	}

	if len(c.TokenCachePaths) == 0 {
		return errTokenCachePathsEmpty
	}

	normalized := make([]string, 0, len(c.TokenCachePaths))
	for _, path := range c.TokenCachePaths {
		normalizedPath, err := normalizePath(path)
		if err != nil {
			return err
		}
		normalized = append(normalized, normalizedPath)
	}

	c.SSO.Region = strings.TrimSpace(c.SSO.Region)
	c.SSO.StartURL = strings.TrimSpace(c.SSO.StartURL)
	if c.SSO.SessionName == "" {
		c.SSO.SessionName = defaultSSOSessionName
	}
	c.TokenCachePaths = normalized

	return nil
}

func defaultSSOConfig() SSOConfig {
	return SSOConfig{
		Region:      "",
		SessionName: defaultSSOSessionName,
		StartURL:    "",
	}
}

func defaultTokenCachePaths() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return []string{}
	}

	return []string{
		filepath.Join(home, ".aws", "sso", "cache"),
		filepath.Join(home, ".granted", "sso"),
	}
}

func expandHomeDir(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("failed to resolve home directory")
	}

	if path == "~" {
		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

func normalizePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errTokenCachePathsEmpty
	}

	expanded, err := expandHomeDir(os.ExpandEnv(path))
	if err != nil {
		return "", err
	}

	expanded = filepath.Clean(expanded)
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("path must be absolute: %s", path)
	}

	return expanded, nil
}
