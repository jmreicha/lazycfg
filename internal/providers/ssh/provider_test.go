package ssh

import (
	"context"
	"testing"

	"github.com/jmreicha/lazycfg/internal/core"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)

	if got := provider.Name(); got != providerName {
		t.Errorf("Name() = %q, want %q", got, providerName)
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
				Enabled:    true,
				ConfigPath: "/home/user/.ssh",
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

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				ConfigPath: "/home/user/.ssh",
				Hosts: []HostConfig{
					{
						Host:     "example.com",
						Hostname: "example.com",
						User:     "user",
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
		},
		{
			name: "empty host pattern",
			config: &Config{
				ConfigPath: "/home/user/.ssh",
				Hosts: []HostConfig{
					{
						Host:     "",
						Hostname: "example.com",
					},
				},
			},
			expectError: true,
		},
		{
			name: "no hosts",
			config: &Config{
				ConfigPath: "/home/user/.ssh",
				Hosts:      []HostConfig{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
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
			name: "valid provider",
			provider: NewProvider(&Config{
				ConfigPath: "/home/user/.ssh",
			}),
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
			name: "invalid config",
			provider: NewProvider(&Config{
				ConfigPath: "",
			}),
			expectError: true,
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
	provider := NewProvider(&Config{
		ConfigPath: "/home/user/.ssh",
	})

	tests := []struct {
		name string
		opts *core.GenerateOptions
	}{
		{
			name: "normal mode",
			opts: &core.GenerateOptions{
				DryRun: false,
				Force:  false,
			},
		},
		{
			name: "dry-run mode",
			opts: &core.GenerateOptions{
				DryRun: true,
			},
		},
		{
			name: "force mode",
			opts: &core.GenerateOptions{
				Force: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := provider.Generate(ctx, tt.opts)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("result is nil")
			}

			if result.Provider != providerName {
				t.Errorf("Provider = %q, want %q", result.Provider, providerName)
			}

			if result.FilesCreated == nil {
				t.Error("FilesCreated is nil")
			}

			if result.FilesSkipped == nil {
				t.Error("FilesSkipped is nil")
			}

			if tt.opts.DryRun && len(result.Warnings) == 0 {
				t.Error("expected warnings in dry-run mode")
			}
		})
	}
}

func TestProvider_Backup(t *testing.T) {
	provider := NewProvider(&Config{
		ConfigPath: "/home/user/.ssh",
	})

	ctx := context.Background()
	backupPath, err := provider.Backup(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Skeleton implementation returns empty string
	if backupPath != "" {
		t.Errorf("backupPath = %q, want empty string", backupPath)
	}
}

func TestProvider_Restore(t *testing.T) {
	provider := NewProvider(&Config{
		ConfigPath: "/home/user/.ssh",
	})

	tests := []struct {
		name        string
		backupPath  string
		expectError bool
	}{
		{
			name:        "empty backup path",
			backupPath:  "",
			expectError: false,
		},
		{
			name:        "non-empty backup path",
			backupPath:  "/tmp/backup",
			expectError: true, // Not yet implemented
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := provider.Restore(ctx, tt.backupPath)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestProvider_Clean(t *testing.T) {
	provider := NewProvider(&Config{
		ConfigPath: "/home/user/.ssh",
	})

	ctx := context.Background()
	err := provider.Clean(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProvider_InterfaceCompliance(_ *testing.T) {
	// Compile-time check that Provider implements core.Provider
	var _ core.Provider = (*Provider)(nil)
}
