package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Verbose {
		t.Error("expected Verbose to be false")
	}
	if cfg.DryRun {
		t.Error("expected DryRun to be false")
	}
	if cfg.NoBackup {
		t.Error("expected NoBackup to be false")
	}
	if cfg.Providers == nil {
		t.Error("expected Providers map to be initialized")
	}
}

func TestLoadConfig_NotExists(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for nonexistent file, got %v", err)
	}

	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
}

func TestLoadConfig_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `verbose: true
dry_run: true
providers:
  aws:
    sso_start_url: https://example.com
`

	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if !cfg.Verbose {
		t.Error("expected Verbose to be true")
	}
	if !cfg.DryRun {
		t.Error("expected DryRun to be true")
	}

	awsCfg := cfg.GetProviderConfig("aws")
	if awsCfg == nil {
		t.Fatal("expected AWS provider config")
	}

	if awsCfg["sso_start_url"] != "https://example.com" {
		t.Errorf("unexpected sso_start_url value: %v", awsCfg["sso_start_url"])
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `invalid: [yaml content`

	if err := os.WriteFile(cfgPath, []byte(invalidYAML), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestConfig_GetProviderConfig(t *testing.T) {
	cfg := NewConfig()
	cfg.SetProviderConfig("test", map[string]interface{}{
		"key": "value",
	})

	providerCfg := cfg.GetProviderConfig("test")
	if providerCfg == nil {
		t.Fatal("expected provider config, got nil")
	}

	if providerCfg["key"] != "value" {
		t.Errorf("unexpected value: %v", providerCfg["key"])
	}
}

func TestConfig_GetProviderConfig_NotExists(t *testing.T) {
	cfg := NewConfig()

	providerCfg := cfg.GetProviderConfig("nonexistent")
	if providerCfg != nil {
		t.Errorf("expected nil for nonexistent provider, got %v", providerCfg)
	}
}

func TestConfig_Merge(t *testing.T) {
	cfg1 := NewConfig()
	cfg1.Verbose = false
	cfg1.SetProviderConfig("aws", map[string]interface{}{
		"region": "us-west-2",
	})

	cfg2 := NewConfig()
	cfg2.Verbose = true
	cfg2.DryRun = true
	cfg2.SetProviderConfig("kubernetes", map[string]interface{}{
		"context": "prod",
	})

	cfg1.Merge(cfg2)

	if !cfg1.Verbose {
		t.Error("expected Verbose to be true after merge")
	}
	if !cfg1.DryRun {
		t.Error("expected DryRun to be true after merge")
	}

	awsCfg := cfg1.GetProviderConfig("aws")
	if awsCfg == nil {
		t.Error("expected AWS config to be preserved")
	}

	k8sCfg := cfg1.GetProviderConfig("kubernetes")
	if k8sCfg == nil {
		t.Error("expected Kubernetes config to be added")
	}
}
