package aws

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmreicha/lazycfg/internal/core"
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

func TestProviderGenerateCredentialsDisabledWhenUseCredentialProcessFalse(t *testing.T) {
	configDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.ConfigPath = filepath.Join(configDir, "config")
	cfg.CredentialsPath = filepath.Join(configDir, "credentials")
	cfg.GenerateCredentials = true
	cfg.UseCredentialProcess = false
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL
	cfg.TokenCachePaths = []string{t.TempDir()}
	provider := NewProvider(cfg)
	provider.discover = func(context.Context, *Config) ([]DiscoveredProfile, error) {
		return []DiscoveredProfile{}, nil
	}

	result, err := provider.Generate(context.Background(), &core.GenerateOptions{Force: true})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if _, err := os.Stat(cfg.CredentialsPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no credentials file, got %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning about credentials generation")
	}
}

func TestProviderGenerateCredentialsWithCredentialProcess(t *testing.T) {
	configDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.ConfigPath = filepath.Join(configDir, "config")
	cfg.CredentialsPath = filepath.Join(configDir, "credentials")
	cfg.GenerateCredentials = true
	cfg.UseCredentialProcess = true
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL
	cfg.TokenCachePaths = []string{t.TempDir()}
	provider := NewProvider(cfg)
	provider.discover = func(context.Context, *Config) ([]DiscoveredProfile, error) {
		return []DiscoveredProfile{
			{
				AccountID:   "111111111111",
				AccountName: "prod",
				RoleName:    "Admin",
			},
		}, nil
	}

	result, err := provider.Generate(context.Background(), &core.GenerateOptions{Force: true})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if _, err := os.Stat(cfg.CredentialsPath); err != nil {
		t.Fatalf("expected credentials file: %v", err)
	}
}

func TestProviderBackupNoFile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfigPath = filepath.Join(t.TempDir(), "nonexistent")
	provider := NewProvider(cfg)

	backup, err := provider.Backup(context.Background())
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if backup != "" {
		t.Fatalf("expected empty backup path, got %q", backup)
	}
}

func TestProviderBackupCreatesBackup(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")
	if err := os.WriteFile(cfgPath, []byte("[default]\nregion = us-east-1\n"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := DefaultConfig()
	cfg.ConfigPath = cfgPath
	provider := NewProvider(cfg)

	backup, err := provider.Backup(context.Background())
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	if !strings.HasPrefix(backup, cfgPath+".") || !strings.HasSuffix(backup, ".bak") {
		t.Fatalf("backup = %q, expected timestamped .bak file", backup)
	}

	data, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(data) != "[default]\nregion = us-east-1\n" {
		t.Fatalf("backup content = %q", string(data))
	}
}

func TestProviderNeedsBackup(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config")
	provider := NewProvider(&Config{ConfigPath: configPath, Enabled: true})

	if needsBackup, err := provider.NeedsBackup(&core.GenerateOptions{DryRun: true}); err != nil {
		t.Fatalf("NeedsBackup error: %v", err)
	} else if needsBackup {
		t.Fatal("expected NeedsBackup to be false for dry-run")
	}

	if err := os.WriteFile(configPath, []byte("data"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if needsBackup, err := provider.NeedsBackup(&core.GenerateOptions{Force: false}); err != nil {
		t.Fatalf("NeedsBackup error: %v", err)
	} else if needsBackup {
		t.Fatal("expected NeedsBackup to be false when config exists without force")
	}

	if needsBackup, err := provider.NeedsBackup(&core.GenerateOptions{Force: true}); err != nil {
		t.Fatalf("NeedsBackup error: %v", err)
	} else if !needsBackup {
		t.Fatal("expected NeedsBackup to be true when forcing overwrite")
	}
}

func TestProviderNeedsBackupDisabled(t *testing.T) {
	provider := NewProvider(&Config{ConfigPath: "/tmp/config", Enabled: false})

	if needsBackup, err := provider.NeedsBackup(&core.GenerateOptions{Force: true}); err != nil {
		t.Fatalf("NeedsBackup error: %v", err)
	} else if needsBackup {
		t.Fatal("expected NeedsBackup to be false when provider is disabled")
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
