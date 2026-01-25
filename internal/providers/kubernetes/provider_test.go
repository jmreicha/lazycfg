package kubernetes

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmreicha/lazycfg/internal/core"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)

	if got := provider.Name(); got != ProviderName {
		t.Errorf("Name() = %q, want %q", got, ProviderName)
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name: "with config",
			config: &Config{
				ConfigPath: "/home/user/.kube/config",
				AWS: AWSConfig{
					Regions:         []string{"us-east-1"},
					ParallelWorkers: 5,
				},
				NamingPattern: "{{.Profile}}-{{.Region}}-{{.Name}}",
				Enabled:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewProvider(tt.config)
			if provider == nil {
				t.Fatal("NewProvider() returned nil")
			}

			if provider.config == nil {
				t.Error("provider.config is nil")
			}
		})
	}
}

func TestProvider_Validate(t *testing.T) {
	tests := []struct {
		name        string
		provider    *Provider
		expectError bool
	}{
		{
			name: "valid provider with demo mode",
			provider: func() *Provider {
				cfg := DefaultConfig()
				cfg.Demo = true
				return NewProvider(cfg)
			}(),
			expectError: false,
		},
		{
			name: "nil config",
			provider: &Provider{
				config: nil,
			},
			expectError: true,
		},
		{
			name: "invalid config - empty path",
			provider: func() *Provider {
				cfg := DefaultConfig()
				cfg.ConfigPath = ""
				cfg.Demo = true
				return NewProvider(cfg)
			}(),
			expectError: true,
		},
		{
			name: "disabled provider - skips validation",
			provider: NewProvider(&Config{
				ConfigPath: "",
				Enabled:    false,
			}),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.provider.Validate(ctx)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestProvider_Generate(t *testing.T) {
	tests := []struct {
		name        string
		provider    *Provider
		opts        *core.GenerateOptions
		expectError bool
	}{
		{
			name: "demo mode",
			provider: func() *Provider {
				cfg := DefaultConfig()
				cfg.ConfigPath = filepath.Join(t.TempDir(), "config")
				cfg.Demo = true
				return NewProvider(cfg)
			}(),
			opts: &core.GenerateOptions{
				Force:  true,
				DryRun: false,
			},
			expectError: false,
		},
		{
			name: "dry-run mode",
			provider: func() *Provider {
				cfg := DefaultConfig()
				cfg.ConfigPath = filepath.Join(t.TempDir(), "config")
				cfg.Demo = true
				return NewProvider(cfg)
			}(),
			opts: &core.GenerateOptions{
				Force:  false,
				DryRun: true,
			},
			expectError: false,
		},
		{
			name: "disabled provider",
			provider: func() *Provider {
				cfg := DefaultConfig()
				cfg.ConfigPath = filepath.Join(t.TempDir(), "config")
				cfg.Enabled = false
				return NewProvider(cfg)
			}(),
			opts: &core.GenerateOptions{
				Force:  false,
				DryRun: false,
			},
			expectError: false,
		},
		{
			name: "merge only mode",
			provider: func() *Provider {
				cfg := DefaultConfig()
				cfg.ConfigPath = filepath.Join(t.TempDir(), "config")
				cfg.MergeOnly = true
				return NewProvider(cfg)
			}(),
			opts: &core.GenerateOptions{
				Force:  true,
				DryRun: false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := tt.provider.Generate(ctx, tt.opts)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil && !tt.expectError {
				t.Error("expected result, got nil")
			}
		})
	}
}

func TestProvider_GenerateInvalidConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		opts     *core.GenerateOptions
	}{
		{
			name: "empty config path",
			provider: func() *Provider {
				cfg := DefaultConfig()
				cfg.ConfigPath = ""
				cfg.Demo = true
				return NewProvider(cfg)
			}(),
			opts: &core.GenerateOptions{
				Force:  true,
				DryRun: false,
			},
		},
		{
			name: "nil config",
			provider: &Provider{
				config: nil,
			},
			opts: &core.GenerateOptions{
				Force:  true,
				DryRun: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := tt.provider.Generate(ctx, tt.opts)

			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestProvider_Backup(t *testing.T) {
	provider := NewProvider(&Config{
		ConfigPath: "/home/user/.kube/config",
		Enabled:    true,
	})

	ctx := context.Background()
	backupPath, err := provider.Backup(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if backupPath != "" {
		t.Errorf("expected empty backup path, got %q", backupPath)
	}
}

func TestProvider_Restore(t *testing.T) {
	provider := NewProvider(&Config{
		ConfigPath: "/home/user/.kube/config",
		Enabled:    true,
	})

	ctx := context.Background()
	err := provider.Restore(ctx, "/path/to/backup")

	if err == nil {
		t.Error("expected error for unimplemented restore, got nil")
	}
}

func TestProvider_Clean(t *testing.T) {
	t.Run("clean existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config")

		// Create a file to clean
		if err := os.WriteFile(configPath, []byte("test"), 0600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		provider := NewProvider(&Config{
			ConfigPath: configPath,
			Enabled:    true,
		})

		ctx := context.Background()
		err := provider.Clean(ctx)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			t.Error("expected file to be removed")
		}
	})

	t.Run("clean non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config")

		provider := NewProvider(&Config{
			ConfigPath: configPath,
			Enabled:    true,
		})

		ctx := context.Background()
		err := provider.Clean(ctx)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		provider := &Provider{
			config: nil,
		}

		ctx := context.Background()
		err := provider.Clean(ctx)

		if err == nil {
			t.Error("expected error for nil config, got nil")
		}
	})
}

func TestProvider_InterfaceCompliance(_ *testing.T) {
	var _ core.Provider = (*Provider)(nil)
}

func TestProvider_ValidateInvalidPaths(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		expectError bool
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
			name:        "absolute path with demo mode",
			configPath:  "/absolute/path",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ConfigPath = tt.configPath
			cfg.Demo = true
			provider := NewProvider(cfg)

			ctx := context.Background()
			err := provider.Validate(ctx)

			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}
