package ssh

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jmreicha/cfgctl/internal/core"
)

const testSSHPath = "/home/user/.ssh"

// TestConfig_Validate tests comprehensive validation scenarios.
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				ConfigPath: testSSHPath,
				GlobalOptions: map[string]string{
					"ServerAliveInterval": "60",
				},
				Hosts: []HostConfig{
					{
						Host:     "example.com",
						Hostname: "example.com",
						User:     "user",
						Port:     22,
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty config path",
			config: &Config{
				ConfigPath: "",
			},
			expectError: true,
			errorMsg:    "config path cannot be empty",
		},
		{
			name: "relative config path",
			config: &Config{
				ConfigPath: "relative/path",
			},
			expectError: true,
			errorMsg:    "must be absolute",
		},
		{
			name: "config with environment variable",
			config: &Config{
				ConfigPath: "$HOME/.ssh",
			},
			expectError: false,
		},
		{
			name: "empty host pattern",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts: []HostConfig{
					{
						Host: "",
					},
				},
			},
			expectError: true,
			errorMsg:    "host pattern cannot be empty",
		},
		{
			name: "invalid port - negative",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts: []HostConfig{
					{
						Host: "test.com",
						Port: -1,
					},
				},
			},
			expectError: true,
			errorMsg:    "port must be between 0 and 65535",
		},
		{
			name: "invalid port - too large",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts: []HostConfig{
					{
						Host: "test.com",
						Port: 65536,
					},
				},
			},
			expectError: true,
			errorMsg:    "port must be between 0 and 65535",
		},
		{
			name: "valid port boundaries",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts: []HostConfig{
					{
						Host: "test1.com",
						Port: 0,
					},
					{
						Host: "test2.com",
						Port: 65535,
					},
				},
			},
			expectError: false,
		},
		{
			name: "relative identity file path",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts: []HostConfig{
					{
						Host:         "test.com",
						IdentityFile: "relative/key",
					},
				},
			},
			expectError: true,
			errorMsg:    "identity file must be an absolute path",
		},
		{
			name: "relative identity agent path",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts: []HostConfig{
					{
						Host:          "test.com",
						IdentityAgent: "relative/agent",
					},
				},
			},
			expectError: true,
			errorMsg:    "identity agent must be an absolute path",
		},
		{
			name: "valid absolute paths",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts: []HostConfig{
					{
						Host:          "test.com",
						IdentityFile:  "/home/user/.ssh/id_rsa",
						IdentityAgent: "/tmp/ssh-agent.sock",
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty hosts list",
			config: &Config{
				ConfigPath: testSSHPath,
				Hosts:      []HostConfig{},
			},
			expectError: false,
		},
		{
			name: "global options only",
			config: &Config{
				ConfigPath: testSSHPath,
				GlobalOptions: map[string]string{
					"ServerAliveInterval": "60",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("error message = %q, want substring %q", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestHostConfig_Validate tests host configuration validation.
func TestHostConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		hostConfig  HostConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid minimal config",
			hostConfig: HostConfig{
				Host: "example.com",
			},
			expectError: false,
		},
		{
			name: "valid complete config",
			hostConfig: HostConfig{
				Host:          "example.com",
				Hostname:      "192.168.1.10",
				User:          "admin",
				Port:          22,
				IdentityFile:  "/home/user/.ssh/id_rsa",
				IdentityAgent: "/tmp/ssh-agent.sock",
				Options: map[string]string{
					"StrictHostKeyChecking": "no",
				},
			},
			expectError: false,
		},
		{
			name: "empty host pattern",
			hostConfig: HostConfig{
				Host: "",
			},
			expectError: true,
			errorMsg:    "host pattern cannot be empty",
		},
		{
			name: "wildcard pattern",
			hostConfig: HostConfig{
				Host: "*.example.com",
			},
			expectError: false,
		},
		{
			name: "complex pattern",
			hostConfig: HostConfig{
				Host: "host?.prod.*.example.com",
			},
			expectError: false,
		},
		{
			name: "negated pattern",
			hostConfig: HostConfig{
				Host: "*.example.com !test.example.com",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hostConfig.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("error message = %q, want substring %q", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestConfigFromMap tests configuration parsing from raw map.
func TestConfigFromMap(t *testing.T) {
	tests := []struct {
		name        string
		raw         map[string]interface{}
		expectError bool
		validate    func(*testing.T, *Config)
	}{
		{
			name:        "nil map",
			raw:         nil,
			expectError: false,
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if c == nil {
					t.Error("config is nil")
				}
			},
		},
		{
			name: "empty map",
			raw:  map[string]interface{}{},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if !c.Enabled {
					t.Error("expected enabled to be true by default")
				}
			},
		},
		{
			name: "valid config",
			raw: map[string]interface{}{
				"enabled":     true,
				"config_path": "/home/user/.ssh",
				"hosts": []interface{}{
					map[string]interface{}{
						"host":     "example.com",
						"hostname": "example.com",
						"user":     "testuser",
						"port":     22,
					},
				},
			},
			validate: func(t *testing.T, c *Config) {
				t.Helper()
				if !c.Enabled {
					t.Error("expected enabled to be true")
				}
				if c.ConfigPath != testSSHPath {
					t.Errorf("config path = %s, want %s", c.ConfigPath, testSSHPath)
				}
				if len(c.Hosts) != 1 {
					t.Errorf("hosts count = %d, want 1", len(c.Hosts))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ConfigFromMap(tt.raw)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

// TestDefaultConfig tests default configuration values.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("default config is nil")
	}

	if !cfg.Enabled {
		t.Error("expected enabled to be true by default")
	}

	if !cfg.ParseHistory {
		t.Error("expected parse_history to be true by default")
	}

	if len(cfg.GlobalOptions) == 0 {
		t.Error("expected default global options")
	}

	if len(cfg.Hosts) != 0 {
		t.Error("expected empty hosts list by default")
	}

	// Verify default config path is set
	if cfg.ConfigPath == "" {
		// OK if home dir is not available
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			t.Error("expected default config path when home is available")
		}
	} else if !contains(cfg.ConfigPath, ".ssh") {
		t.Errorf("expected config path to contain .ssh, got %s", cfg.ConfigPath)
	}
}

// TestConfig_NormalizedConfigPath tests path normalization.
func TestConfig_NormalizedConfigPath(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		expectError bool
		validate    func(*testing.T, string)
	}{
		{
			name:        "empty path",
			configPath:  "",
			expectError: true,
		},
		{
			name:        "relative path",
			configPath:  "relative/path",
			expectError: true,
		},
		{
			name:       "absolute path",
			configPath: testSSHPath,
			validate: func(t *testing.T, result string) {
				t.Helper()
				if result != testSSHPath {
					t.Errorf("normalized path = %s, want %s", result, testSSHPath)
				}
			},
		},
		{
			name:       "path with trailing slash",
			configPath: testSSHPath + "/",
			validate: func(t *testing.T, result string) {
				t.Helper()
				if result != testSSHPath {
					t.Errorf("normalized path = %s, want %s", result, testSSHPath)
				}
			},
		},
		{
			name:       "path with multiple slashes",
			configPath: "/home//user/.ssh",
			validate: func(t *testing.T, result string) {
				t.Helper()
				if result != testSSHPath {
					t.Errorf("normalized path = %s, want %s", result, testSSHPath)
				}
			},
		},
		{
			name:       "path with environment variable",
			configPath: "$HOME/.ssh",
			validate: func(t *testing.T, result string) {
				t.Helper()
				home := os.Getenv("HOME")
				if home != "" {
					expected := filepath.Join(home, ".ssh")
					if result != expected {
						t.Errorf("normalized path = %s, want %s", result, expected)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ConfigPath: tt.configPath,
			}

			result, err := cfg.normalizedConfigPath()

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConfigFromMapInvalidYAML(t *testing.T) {
	raw := map[string]interface{}{
		"enabled": "not-a-bool",
	}

	_, err := ConfigFromMap(raw)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestDefaultConfigPathNoHome(t *testing.T) {
	t.Setenv("HOME", "")

	path := defaultConfigPath()
	if path != "" {
		t.Errorf("defaultConfigPath() = %q, want empty string", path)
	}
}

func TestProviderConfigFactoryRegistration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := core.ProviderConfigFromMap("ssh", map[string]interface{}{})
	if err != nil {
		t.Fatalf("ProviderConfigFromMap failed: %v", err)
	}

	sshCfg, ok := cfg.(*Config)
	if !ok {
		t.Fatalf("expected *Config, got %T", cfg)
	}

	if !sshCfg.Enabled {
		t.Error("expected enabled to be true by default")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
