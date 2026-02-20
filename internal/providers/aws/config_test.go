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
	if cfg.CredentialsPath != filepath.Join(home, ".aws", "credentials") {
		t.Fatalf("credentials path = %q", cfg.CredentialsPath)
	}
}

func TestConfigFromMapOverrides(t *testing.T) {
	raw := map[string]interface{}{
		"credentials_path":     "/custom/credentials",
		"generate_credentials": true,
		"roles":                []interface{}{"Admin"},
		"sso": map[string]interface{}{
			"region":       testRegion,
			"session_name": "custom",
			"start_url":    testStartURL,
		},
		"token_cache_paths":      []interface{}{`/cache`},
		"use_credential_process": true,
		"marker_key":             "custom-marker",
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
	if cfg.CredentialsPath != "/custom/credentials" {
		t.Fatalf("credentials path = %q", cfg.CredentialsPath)
	}
	if !cfg.GenerateCredentials {
		t.Fatal("expected generate credentials enabled")
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
	if !cfg.UseCredentialProcess {
		t.Fatal("expected credential process enabled")
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

func TestConfigFromMapNil(t *testing.T) {
	cfg, err := ConfigFromMap(nil)
	if err != nil {
		t.Fatalf("ConfigFromMap(nil) failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestConfigFromMapEmpty(t *testing.T) {
	cfg, err := ConfigFromMap(map[string]interface{}{})
	if err != nil {
		t.Fatalf("ConfigFromMap(empty) failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
		return
	}
	if cfg.ProfileTemplate != defaultProfileTemplate {
		t.Fatalf("expected default profile template, got %q", cfg.ProfileTemplate)
	}
	if cfg.MarkerKey != defaultMarkerKey {
		t.Fatalf("expected default marker key, got %q", cfg.MarkerKey)
	}
	if cfg.SSO.RegistrationScopes != defaultSSOScopes {
		t.Fatalf("expected default scopes, got %q", cfg.SSO.RegistrationScopes)
	}
	if cfg.SSO.SessionName != defaultSSOSessionName {
		t.Fatalf("expected default session name, got %q", cfg.SSO.SessionName)
	}
}

func TestConfigFromMapWithProfileTemplate(t *testing.T) {
	raw := map[string]interface{}{
		"profile_template": "{{ .AccountName }}-{{ .RoleName }}",
		"sso": map[string]interface{}{
			"region":    testRegion,
			"start_url": testStartURL,
		},
	}
	cfg, err := ConfigFromMap(raw)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}
	if cfg.ProfileTemplate != "{{ .AccountName }}-{{ .RoleName }}" {
		t.Fatalf("profile template = %q", cfg.ProfileTemplate)
	}
}

func TestConfigFromMapWithRoleChains(t *testing.T) {
	raw := map[string]interface{}{
		"sso": map[string]interface{}{
			"region":    testRegion,
			"start_url": testStartURL,
		},
		"role_chains": []interface{}{
			map[string]interface{}{
				"name":           "chained",
				"role_arn":       "arn:aws:iam::123456789012:role/Test",
				"source_profile": "source",
			},
		},
	}
	cfg, err := ConfigFromMap(raw)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}
	if len(cfg.RoleChains) != 1 {
		t.Fatalf("expected 1 role chain, got %d", len(cfg.RoleChains))
	}
	if cfg.RoleChains[0].Name != "chained" {
		t.Fatalf("role chain name = %q", cfg.RoleChains[0].Name)
	}
}

func TestConfigFromMapWithEmptyMarkerKey(t *testing.T) {
	raw := map[string]interface{}{
		"marker_key": "   ",
		"sso": map[string]interface{}{
			"region":    testRegion,
			"start_url": testStartURL,
		},
	}
	cfg, err := ConfigFromMap(raw)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}
	if cfg.MarkerKey != defaultMarkerKey {
		t.Fatalf("expected default marker key for whitespace input, got %q", cfg.MarkerKey)
	}
}

func TestNormalizeConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "whitespace path",
			path:    "   ",
			wantErr: true,
		},
		{
			name:    "relative path",
			path:    "relative/config",
			wantErr: true,
		},
		{
			name:    "absolute path",
			path:    "/tmp/config",
			wantErr: false,
		},
		{
			name:    "tilde path",
			path:    "~/config",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeConfigPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeConfigPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestConfigIsEnabled(t *testing.T) {
	var nilCfg *Config
	if nilCfg.IsEnabled() {
		t.Fatal("expected nil config to not be enabled")
	}

	cfg := &Config{Enabled: true}
	if !cfg.IsEnabled() {
		t.Fatal("expected config to be enabled")
	}

	cfg.Enabled = false
	if cfg.IsEnabled() {
		t.Fatal("expected config to not be enabled")
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
			name: "missing credentials path",
			cfg: func() *Config {
				cfg := *base
				cfg.GenerateCredentials = true
				cfg.CredentialsPath = ""
				return &cfg
			}(),
		},
		{
			name: "relative credentials path",
			cfg: func() *Config {
				cfg := *base
				cfg.GenerateCredentials = true
				cfg.CredentialsPath = "relative"
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
		{
			name: "demo mode skips token validation",
			cfg: func() *Config {
				cfg := *base
				cfg.Demo = true
				cfg.TokenCachePaths = nil
				cfg.SSO.StartURL = ""
				cfg.SSO.Region = ""
				return &cfg
			}(),
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
			err := tt.cfg.Validate()
			if tt.name == "demo mode skips token validation" {
				if err != nil {
					t.Fatalf("expected no error for %s, got %v", tt.name, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}
