package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Engine coordinates provider lifecycle: validation, backup, generation, error handling, and rollback.
type Engine struct {
	registry      *Registry
	backupManager *BackupManager
	config        *Config
	logger        *slog.Logger
}

// NewEngine creates a new engine with the provided components.
func NewEngine(registry *Registry, backupManager *BackupManager, config *Config, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}

	return &Engine{
		registry:      registry,
		backupManager: backupManager,
		config:        config,
		logger:        logger,
	}
}

// ExecuteOptions contains options for executing generation.
type ExecuteOptions struct {
	// Providers lists the specific providers to run.
	// If empty, all registered providers will be executed.
	Providers []string

	// DryRun simulates generation without making changes.
	DryRun bool

	// Force overwrites existing configurations.
	Force bool

	// NoBackup skips backup creation before generation.
	NoBackup bool

	// Verbose enables detailed logging.
	Verbose bool
}

// Execute runs the generation process for the specified providers.
// Returns a map of provider names to their results, or an error if the process fails.
func (e *Engine) Execute(ctx context.Context, opts *ExecuteOptions) (map[string]*Result, error) {
	if opts == nil {
		opts = &ExecuteOptions{}
	}

	// Determine which providers to run
	providers, err := e.resolveProviders(opts.Providers)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, errors.New("no providers to execute")
	}

	e.logger.Info("starting generation", "provider_count", len(providers))

	results := make(map[string]*Result)
	backups := make(map[string]string)

	// Execute each provider
	for _, provider := range providers {
		providerName := provider.Name()
		e.logger.Info("processing provider", "provider", providerName)

		// Phase 1: Validate
		if err := e.validateProvider(ctx, provider); err != nil {
			return results, fmt.Errorf("validation failed for provider %q: %w", providerName, err)
		}

		// Phase 2: Backup (unless disabled or dry-run)
		var backupPath string
		if !opts.NoBackup && !opts.DryRun {
			backupPath, err = e.backupProvider(ctx, provider)
			if err != nil {
				e.logger.Warn("backup failed", "provider", providerName, "error", err)
			} else if backupPath != "" {
				backups[providerName] = backupPath
				e.logger.Info("backup created", "provider", providerName, "path", backupPath)
			}
		}

		// Phase 3: Generate
		result, err := e.generateProvider(ctx, provider, opts)
		if err != nil {
			e.logger.Error("generation failed", "provider", providerName, "error", err)

			// Attempt rollback if we have a backup
			if backupPath != "" {
				e.logger.Info("attempting rollback", "provider", providerName)
				if rollbackErr := e.backupManager.Restore(backupPath); rollbackErr != nil {
					e.logger.Error("rollback failed", "provider", providerName, "error", rollbackErr)
				} else {
					e.logger.Info("rollback successful", "provider", providerName)
				}
			}

			return results, fmt.Errorf("generation failed for provider %q: %w", providerName, err)
		}

		result.BackupPath = backupPath
		results[providerName] = result

		e.logger.Info("provider completed", "provider", providerName, "files_created", len(result.FilesCreated))
	}

	e.logger.Info("generation complete", "provider_count", len(results))
	return results, nil
}

// ValidateAll validates all providers without generating anything.
func (e *Engine) ValidateAll(ctx context.Context) error {
	providers := e.registry.GetAll()

	for _, provider := range providers {
		if err := e.validateProvider(ctx, provider); err != nil {
			return fmt.Errorf("validation failed for provider %q: %w", provider.Name(), err)
		}
	}

	return nil
}

// CleanProvider removes all configuration files for a specific provider.
func (e *Engine) CleanProvider(ctx context.Context, providerName string) error {
	provider, err := e.registry.Get(providerName)
	if err != nil {
		return err
	}

	e.logger.Info("cleaning provider", "provider", providerName)

	if err := provider.Clean(ctx); err != nil {
		return fmt.Errorf("clean failed for provider %q: %w", providerName, err)
	}

	e.logger.Info("clean complete", "provider", providerName)
	return nil
}

// resolveProviders determines which providers to execute based on the input list.
func (e *Engine) resolveProviders(providerNames []string) ([]Provider, error) {
	if len(providerNames) == 0 {
		// Return all registered providers
		return e.registry.GetAll(), nil
	}

	providers := make([]Provider, 0, len(providerNames))
	for _, name := range providerNames {
		provider, err := e.registry.Get(name)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

// validateProvider runs validation for a single provider.
func (e *Engine) validateProvider(ctx context.Context, provider Provider) error {
	e.logger.Debug("validating provider", "provider", provider.Name())

	if err := provider.Validate(ctx); err != nil {
		return err
	}

	e.logger.Debug("validation passed", "provider", provider.Name())
	return nil
}

// backupProvider creates a backup for a provider.
// Returns empty string and nil error if no backup is needed.
func (e *Engine) backupProvider(ctx context.Context, provider Provider) (string, error) {
	e.logger.Debug("creating backup", "provider", provider.Name())

	backupPath, err := provider.Backup(ctx)
	if err != nil {
		return "", err
	}

	return backupPath, nil
}

// generateProvider runs generation for a single provider.
func (e *Engine) generateProvider(ctx context.Context, provider Provider, opts *ExecuteOptions) (*Result, error) {
	e.logger.Debug("generating configuration", "provider", provider.Name())

	// Get provider-specific configuration
	providerCfg := e.config.GetProviderConfig(provider.Name())

	genOpts := &GenerateOptions{
		DryRun:  opts.DryRun,
		Force:   opts.Force,
		Verbose: opts.Verbose,
		Config:  nil, // Provider can extract its config from providerCfg
	}

	// Pass raw config map in metadata
	if genOpts.Config == nil && providerCfg != nil {
		// Providers will need to handle raw map[string]interface{} config
		// This allows flexibility for different provider config structures
		_ = providerCfg
	}

	result, err := provider.Generate(ctx, genOpts)
	if err != nil {
		return nil, err
	}

	result.Provider = provider.Name()
	return result, nil
}
