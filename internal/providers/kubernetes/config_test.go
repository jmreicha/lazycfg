package kubernetes

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jmreicha/lazycfg/internal/core"
)

func TestConfigFromMapDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := ConfigFromMap(map[string]interface{}{})
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}

	if cfg.ConfigPath != filepath.Join(home, ".kube", "config") {
		t.Errorf("ConfigPath = %q", cfg.ConfigPath)
	}

	if cfg.AWS.CredentialsFile != filepath.Join(home, ".aws", "credentials") {
		t.Errorf("CredentialsFile = %q", cfg.AWS.CredentialsFile)
	}

	if !reflect.DeepEqual(cfg.AWS.Regions, defaultRegions()) {
		t.Errorf("Regions = %v", cfg.AWS.Regions)
	}

	if cfg.AWS.ParallelWorkers != defaultParallelWorkers() {
		t.Errorf("ParallelWorkers = %d", cfg.AWS.ParallelWorkers)
	}

	if cfg.AWS.Timeout != defaultTimeout() {
		t.Errorf("Timeout = %v", cfg.AWS.Timeout)
	}

	if cfg.NamingPattern != defaultNamingPattern() {
		t.Errorf("NamingPattern = %q", cfg.NamingPattern)
	}

	if cfg.Merge.SourceDir != filepath.Join(home, ".kube") {
		t.Errorf("Merge.SourceDir = %q", cfg.Merge.SourceDir)
	}

	if !reflect.DeepEqual(cfg.Merge.IncludePatterns, defaultIncludePatterns()) {
		t.Errorf("IncludePatterns = %v", cfg.Merge.IncludePatterns)
	}

	if !reflect.DeepEqual(cfg.Merge.ExcludePatterns, defaultExcludePatterns()) {
		t.Errorf("ExcludePatterns = %v", cfg.Merge.ExcludePatterns)
	}
}

func TestConfigFromMapOverrides(t *testing.T) {
	raw := map[string]interface{}{
		"config_path":    "/custom/kubeconfig",
		"naming_pattern": "{profile}-{region}-{cluster}",
		"aws": map[string]interface{}{
			"credentials_file": "/custom/creds",
			"profiles":         []string{"prod"},
			"regions":          []string{"us-west-2"},
			"parallel_workers": 4,
			"timeout":          "45s",
		},
		"merge": map[string]interface{}{
			"source_dir":       "/custom/merge",
			"include_patterns": []string{"*.yaml"},
			"exclude_patterns": []string{"*.bak"},
		},
	}

	cfg, err := ConfigFromMap(raw)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}

	if cfg.ConfigPath != "/custom/kubeconfig" {
		t.Errorf("ConfigPath = %q", cfg.ConfigPath)
	}

	if cfg.NamingPattern != "{profile}-{region}-{cluster}" {
		t.Errorf("NamingPattern = %q", cfg.NamingPattern)
	}

	if cfg.AWS.CredentialsFile != "/custom/creds" {
		t.Errorf("CredentialsFile = %q", cfg.AWS.CredentialsFile)
	}

	if !reflect.DeepEqual(cfg.AWS.Profiles, []string{"prod"}) {
		t.Errorf("Profiles = %v", cfg.AWS.Profiles)
	}

	if !reflect.DeepEqual(cfg.AWS.Regions, []string{"us-west-2"}) {
		t.Errorf("Regions = %v", cfg.AWS.Regions)
	}

	if cfg.AWS.ParallelWorkers != 4 {
		t.Errorf("ParallelWorkers = %d", cfg.AWS.ParallelWorkers)
	}

	if cfg.AWS.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v", cfg.AWS.Timeout)
	}

	if cfg.Merge.SourceDir != "/custom/merge" {
		t.Errorf("Merge.SourceDir = %q", cfg.Merge.SourceDir)
	}

	if !reflect.DeepEqual(cfg.Merge.IncludePatterns, []string{"*.yaml"}) {
		t.Errorf("IncludePatterns = %v", cfg.Merge.IncludePatterns)
	}

	if !reflect.DeepEqual(cfg.Merge.ExcludePatterns, []string{"*.bak"}) {
		t.Errorf("ExcludePatterns = %v", cfg.Merge.ExcludePatterns)
	}
}

func TestConfigValidate(t *testing.T) {
	baseDir := t.TempDir()

	cfg := &Config{
		Enabled:    true,
		ConfigPath: filepath.Join(baseDir, "kube", "config"),
		AWS: AWSConfig{
			CredentialsFile: filepath.Join(baseDir, "aws", "credentials"),
			Profiles:        []string{"dev"},
			Regions:         []string{"us-west-2"},
			ParallelWorkers: 2,
			Timeout:         5 * time.Second,
		},
		NamingPattern: " {profile}-{cluster} ",
		MergeOnly:     true,
		Merge: MergeConfig{
			SourceDir:       filepath.Join(baseDir, "kube"),
			IncludePatterns: []string{"*.yaml"},
			ExcludePatterns: []string{"*.bak"},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if !cfg.MergeEnabled {
		t.Error("expected MergeEnabled to be true when MergeOnly is set")
	}

	if cfg.NamingPattern != "{profile}-{cluster}" {
		t.Errorf("NamingPattern = %q", cfg.NamingPattern)
	}
}

func TestConfigFromMapNil(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := ConfigFromMap(nil)
	if err != nil {
		t.Fatalf("ConfigFromMap failed: %v", err)
	}

	if cfg == nil {
		t.Error("expected non-nil config")
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

func TestProviderConfigFactoryRegistration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := core.ProviderConfigFromMap("kubernetes", map[string]interface{}{})
	if err != nil {
		t.Fatalf("ProviderConfigFromMap failed: %v", err)
	}

	kubeCfg, ok := cfg.(*Config)
	if !ok {
		t.Fatalf("expected *Config, got %T", cfg)
	}

	if !kubeCfg.Enabled {
		t.Error("expected enabled to be true by default")
	}
}

func TestConfigValidateErrors(t *testing.T) {
	baseDir := t.TempDir()
	base := &Config{
		Enabled:    true,
		ConfigPath: filepath.Join(baseDir, "kube", "config"),
		AWS: AWSConfig{
			CredentialsFile: filepath.Join(baseDir, "aws", "credentials"),
			Regions:         []string{"us-west-2"},
			ParallelWorkers: 1,
			Timeout:         5 * time.Second,
		},
		NamingPattern: defaultNamingPattern(),
		Merge: MergeConfig{
			SourceDir: filepath.Join(baseDir, "kube"),
		},
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
		{
			name: "empty regions",
			mutate: func(cfg *Config) {
				cfg.AWS.Regions = nil
			},
			wantErr: errRegionsEmpty,
		},
		{
			name: "invalid parallel workers",
			mutate: func(cfg *Config) {
				cfg.AWS.ParallelWorkers = 0
			},
			wantErr: errParallelWorkersBounds,
		},
		{
			name: "invalid timeout",
			mutate: func(cfg *Config) {
				cfg.AWS.Timeout = 0
			},
			wantErr: errTimeoutInvalid,
		},
		{
			name: "empty naming pattern",
			mutate: func(cfg *Config) {
				cfg.NamingPattern = "  "
			},
			wantErr: errNamingPatternEmpty,
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
