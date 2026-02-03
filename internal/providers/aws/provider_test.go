package aws

import (
	"context"
	"path/filepath"
	"testing"
)

func TestProviderName(t *testing.T) {
	provider := NewProvider(nil)
	if provider.Name() != ProviderName {
		t.Fatalf("name = %q", provider.Name())
	}
}

func TestProviderValidateDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	provider := NewProvider(cfg)

	if err := provider.Validate(context.Background()); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestProviderGenerateDisabled(t *testing.T) {
	cacheDir := t.TempDir()
	configDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.ConfigPath = filepath.Join(configDir, "config")
	cfg.Enabled = false
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL
	cfg.TokenCachePaths = []string{cacheDir}
	provider := NewProvider(cfg)

	result, err := provider.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
}

func TestProviderBackup(t *testing.T) {
	provider := NewProvider(DefaultConfig())

	backup, err := provider.Backup(context.Background())
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if backup != "" {
		t.Fatalf("backup = %q", backup)
	}
}

func TestProviderRestore(t *testing.T) {
	provider := NewProvider(DefaultConfig())

	if err := provider.Restore(context.Background(), "path"); err == nil {
		t.Fatal("expected restore error")
	}
}

func TestProviderCleanEmptyPath(t *testing.T) {
	provider := NewProvider(&Config{ConfigPath: ""})

	if err := provider.Clean(context.Background()); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}
}
