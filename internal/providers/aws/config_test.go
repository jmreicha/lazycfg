package aws

import (
	"path/filepath"
	"reflect"
	"testing"
)

const (
	testRegion   = "us-east-1"
	testStartURL = "https://example.awsapps.com/start"
)

func TestDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("expected default config")
	}

	if cfg.SSO.SessionName != defaultSSOSessionName {
		t.Fatalf("session name = %q", cfg.SSO.SessionName)
	}
	if cfg.SSO.RegistrationScopes != defaultSSOScopes {
		t.Fatalf("registration scopes = %q", cfg.SSO.RegistrationScopes)
	}
	if cfg.ProfileTemplate != defaultProfileTemplate {
		t.Fatalf("profile template = %q", cfg.ProfileTemplate)
	}
	if cfg.MarkerKey != defaultMarkerKey {
		t.Fatalf("marker key = %q", cfg.MarkerKey)
	}

	expectedPaths := []string{
		filepath.Join(home, ".aws", "sso", "cache"),
		filepath.Join(home, ".granted"),
	}
	if !reflect.DeepEqual(cfg.TokenCachePaths, expectedPaths) {
		t.Fatalf("token cache paths = %#v", cfg.TokenCachePaths)
	}

	if cfg.ConfigPath != filepath.Join(home, ".aws", "config") {
		t.Fatalf("config path = %q", cfg.ConfigPath)
	}
}

func TestConfigFromMapOverrides(t *testing.T) {
	raw := map[string]interface{}{
		"roles": []interface{}{"Admin"},
		"sso": map[string]interface{}{
			"region":       testRegion,
			"session_name": "custom",
			"start_url":    testStartURL,
		},
		"token_cache_paths": []interface{}{`/cache`},
		"marker_key":        "custom-marker",
	}

	cfg, err := ConfigFromMap(raw)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}

	if cfg.SSO.SessionName != "custom" {
		t.Fatalf("session name = %q", cfg.SSO.SessionName)
	}
	if cfg.SSO.RegistrationScopes != defaultSSOScopes {
		t.Fatalf("registration scopes = %q", cfg.SSO.RegistrationScopes)
	}
	if cfg.SSO.Region != testRegion {
		t.Fatalf("region = %q", cfg.SSO.Region)
	}
	if cfg.SSO.StartURL != testStartURL {
		t.Fatalf("start url = %q", cfg.SSO.StartURL)
	}
	if cfg.ProfileTemplate != defaultProfileTemplate {
		t.Fatalf("profile template = %q", cfg.ProfileTemplate)
	}
	if cfg.MarkerKey != "custom-marker" {
		t.Fatalf("marker key = %q", cfg.MarkerKey)
	}
	if !reflect.DeepEqual(cfg.Roles, []string{"Admin"}) {
		t.Fatalf("roles = %#v", cfg.Roles)
	}
	if !reflect.DeepEqual(cfg.TokenCachePaths, []string{"/cache"}) {
		t.Fatalf("token cache paths = %#v", cfg.TokenCachePaths)
	}
}

func TestConfigValidate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := DefaultConfig()
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL
	cfg.TokenCachePaths = []string{"~/.aws/sso/cache"}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	expected := filepath.Join(home, ".aws", "sso", "cache")
	if cfg.TokenCachePaths[0] != expected {
		t.Fatalf("normalized path = %q", cfg.TokenCachePaths[0])
	}

	if cfg.ConfigPath != filepath.Join(home, ".aws", "config") {
		t.Fatalf("config path = %q", cfg.ConfigPath)
	}
}

func TestConfigValidateErrors(t *testing.T) {
	base := DefaultConfig()
	base.SSO.Region = testRegion
	base.SSO.StartURL = testStartURL
	base.TokenCachePaths = []string{"/cache"}

	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "missing start url",
			cfg: func() *Config {
				cfg := *base
				cfg.SSO.StartURL = ""
				return &cfg
			}(),
		},
		{
			name: "missing region",
			cfg: func() *Config {
				cfg := *base
				cfg.SSO.Region = ""
				return &cfg
			}(),
		},
		{
			name: "empty cache paths",
			cfg: func() *Config {
				cfg := *base
				cfg.TokenCachePaths = nil
				return &cfg
			}(),
		},
		{
			name: "relative cache path",
			cfg: func() *Config {
				cfg := *base
				cfg.TokenCachePaths = []string{"relative"}
				return &cfg
			}(),
		},
		{
			name: "config nil",
			cfg:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg == nil {
				var cfg *Config
				if err := cfg.Validate(); err == nil {
					t.Fatalf("expected error for %s", tt.name)
				}
				return
			}
			if err := tt.cfg.Validate(); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}
