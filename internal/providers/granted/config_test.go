package granted

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/jmreicha/lazycfg/internal/core"
)

func TestConfigFromMapDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := ConfigFromMap(map[string]interface{}{})
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}

	if cfg.ConfigPath != filepath.Join(home, ".granted", "config") {
		t.Errorf("ConfigPath = %q", cfg.ConfigPath)
	}

	if !cfg.CredentialProcessAutoLogin {
		t.Error("CredentialProcessAutoLogin should be true by default")
	}

	if cfg.DefaultBrowser != defaultBrowserValue {
		t.Errorf("DefaultBrowser = %q", cfg.DefaultBrowser)
	}

	if !cfg.DisableUsageTips {
		t.Error("DisableUsageTips should be true by default")
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true by default")
	}
}

func TestConfigFromMapNil(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := ConfigFromMap(nil)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}

	if cfg.ConfigPath != filepath.Join(home, ".granted", "config") {
		t.Errorf("ConfigPath = %q", cfg.ConfigPath)
	}
}

func TestConfigFromMapOverrides(t *testing.T) {
	raw := map[string]interface{}{
		"config_path":                   "/custom/granted/config",
		"credential_process_auto_login": false,
		"custom_browser_path":           "/usr/bin/firefox",
		"custom_sso_browser_path":       "/usr/bin/chrome",
		"default_browser":               "FIREFOX",
		"disable_usage_tips":            false,
		"export_credential_suffix":      "_CUSTOM",
		"ordering":                      "alphabetical",
	}

	cfg, err := ConfigFromMap(raw)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}

	if cfg.ConfigPath != "/custom/granted/config" {
		t.Errorf("ConfigPath = %q", cfg.ConfigPath)
	}

	if cfg.CredentialProcessAutoLogin {
		t.Error("CredentialProcessAutoLogin should be false")
	}

	if cfg.CustomBrowserPath != "/usr/bin/firefox" {
		t.Errorf("CustomBrowserPath = %q", cfg.CustomBrowserPath)
	}

	if cfg.CustomSSOBrowserPath != "/usr/bin/chrome" {
		t.Errorf("CustomSSOBrowserPath = %q", cfg.CustomSSOBrowserPath)
	}

	if cfg.DefaultBrowser != "FIREFOX" {
		t.Errorf("DefaultBrowser = %q", cfg.DefaultBrowser)
	}

	if cfg.DisableUsageTips {
		t.Error("DisableUsageTips should be false")
	}

	if cfg.ExportCredentialSuffix != "_CUSTOM" {
		t.Errorf("ExportCredentialSuffix = %q", cfg.ExportCredentialSuffix)
	}

	if cfg.Ordering != "alphabetical" {
		t.Errorf("Ordering = %q", cfg.Ordering)
	}
}

func TestConfigValidate(t *testing.T) {
	baseDir := t.TempDir()

	cfg := &Config{
		ConfigPath:                 filepath.Join(baseDir, "granted", "config"),
		CredentialProcessAutoLogin: true,
		DefaultBrowser:             "STDOUT",
		DisableUsageTips:           true,
		Enabled:                    true,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if cfg.ConfigPath != filepath.Join(baseDir, "granted", "config") {
		t.Errorf("ConfigPath = %q", cfg.ConfigPath)
	}
}

func TestConfigValidateErrors(t *testing.T) {
	baseDir := t.TempDir()
	base := &Config{
		ConfigPath:                 filepath.Join(baseDir, "granted", "config"),
		CredentialProcessAutoLogin: true,
		DefaultBrowser:             "STDOUT",
		DisableUsageTips:           true,
		Enabled:                    true,
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr error
	}{
		{
			name: "empty config path",
			mutate: func(cfg *Config) {
				cfg.ConfigPath = ""
			},
			wantErr: errConfigPathEmpty,
		},
		{
			name: "relative config path",
			mutate: func(cfg *Config) {
				cfg.ConfigPath = "relative/config"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := *base
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestConfigValidateExpandsHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		ConfigPath:     "~/.granted/config",
		DefaultBrowser: "STDOUT",
		Enabled:        true,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	expected := filepath.Join(home, ".granted", "config")
	if cfg.ConfigPath != expected {
		t.Errorf("ConfigPath = %q, want %q", cfg.ConfigPath, expected)
	}
}

func TestConfigValidateExpandsEnvVars(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("GRANTED_DIR", baseDir)

	cfg := &Config{
		ConfigPath:     "$GRANTED_DIR/config",
		DefaultBrowser: "STDOUT",
		Enabled:        true,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	expected := filepath.Join(baseDir, "config")
	if cfg.ConfigPath != expected {
		t.Errorf("ConfigPath = %q, want %q", cfg.ConfigPath, expected)
	}
}

func TestProviderConfigFactoryRegistration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := core.ProviderConfigFromMap("granted", map[string]interface{}{})
	if err != nil {
		t.Fatalf("ProviderConfigFromMap failed: %v", err)
	}

	grantedCfg, ok := cfg.(*Config)
	if !ok {
		t.Fatalf("expected *Config, got %T", cfg)
	}

	if grantedCfg.ConfigPath != filepath.Join(home, ".granted", "config") {
		t.Errorf("ConfigPath = %q", grantedCfg.ConfigPath)
	}

	if grantedCfg.DefaultBrowser != defaultBrowserValue {
		t.Errorf("DefaultBrowser = %q", grantedCfg.DefaultBrowser)
	}
}

func TestConfigFromMapInvalidYAMLUnmarshal(t *testing.T) {
	raw := map[string]interface{}{
		"enabled": "not-a-bool",
	}

	_, err := ConfigFromMap(raw)
	if err == nil {
		t.Error("expected error for invalid YAML unmarshal")
	}
}

func TestDefaultConfigPathWithNoHome(t *testing.T) {
	t.Setenv("HOME", "")

	path := defaultConfigPath()
	if path != "" {
		t.Errorf("defaultConfigPath() = %q, want empty string", path)
	}
}

func TestExpandHomeDirErrors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		homeEnv string
		wantErr bool
	}{
		{
			name:    "empty home directory",
			path:    "~/config",
			homeEnv: "",
			wantErr: true,
		},
		{
			name:    "non-tilde path",
			path:    "/absolute/path",
			homeEnv: "/home/user",
			wantErr: false,
		},
		{
			name:    "tilde only",
			path:    "~",
			homeEnv: "/home/user",
			wantErr: false,
		},
		{
			name:    "tilde with slash",
			path:    "~/config",
			homeEnv: "/home/user",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", tt.homeEnv)

			result, err := expandHomeDir(tt.path)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && tt.homeEnv == "" && result != tt.path {
				t.Errorf("expandHomeDir() = %q, want %q", result, tt.path)
			}
		})
	}
}

func TestNormalizePathErrors(t *testing.T) {
	testErr := errors.New("test error")

	tests := []struct {
		name     string
		path     string
		emptyErr error
		wantErr  bool
	}{
		{
			name:     "empty path",
			path:     "",
			emptyErr: testErr,
			wantErr:  true,
		},
		{
			name:     "whitespace only",
			path:     "   ",
			emptyErr: testErr,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizePath(tt.path, tt.emptyErr)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && !errors.Is(err, tt.emptyErr) {
				t.Errorf("error = %v, want %v", err, tt.emptyErr)
			}
		})
	}
}
