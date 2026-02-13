// Package kubernetes provides a Kubernetes configuration provider implementation.
package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmreicha/lazycfg/internal/core"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ProviderName is the unique identifier for the Kubernetes provider.
const ProviderName = "kubernetes"

var errProviderConfigNil = errors.New("kubernetes provider configuration is nil")

// Provider implements the core.Provider interface for Kubernetes configuration management.
type Provider struct {
	config *Config
}

// NewProvider creates a new Kubernetes provider instance with the given configuration.
func NewProvider(config *Config) *Provider {
	if config == nil {
		config = DefaultConfig()
	}

	return &Provider{config: config}
}

// Name returns the unique identifier for this provider.
func (p *Provider) Name() string {
	return ProviderName
}

// Validate checks if all prerequisites for this provider are met.
func (p *Provider) Validate(_ context.Context) error {
	if p.config == nil {
		return errProviderConfigNil
	}

	if !p.config.Enabled {
		return nil
	}

	return p.config.Validate()
}

// Generate creates the configuration files for this provider.
func (p *Provider) Generate(ctx context.Context, opts *core.GenerateOptions) (*core.Result, error) {
	result := &core.Result{
		Provider:     p.Name(),
		FilesCreated: []string{},
		FilesSkipped: []string{},
		Warnings:     []string{},
		Metadata:     make(map[string]interface{}),
	}

	if err := p.validateAndPrepare(opts); err != nil {
		return nil, err
	}

	if !p.config.Enabled {
		result.Warnings = append(result.Warnings, "kubernetes provider is disabled")
		return result, nil
	}

	discovered, err := p.discoverClusters(ctx)
	if err != nil {
		return nil, err
	}

	mergeConfig, mergeFiles, err := p.buildKubeconfig(discovered)
	if err != nil {
		return nil, err
	}

	if mergeConfig == nil {
		result.Warnings = append(result.Warnings, "no kubeconfig data generated")
		return result, nil
	}

	if len(mergeFiles) > 0 {
		result.Metadata["merge_files"] = mergeFiles
	}

	if opts != nil && opts.DryRun {
		return p.handleDryRun(result, discovered, mergeConfig), nil
	}

	outputPath := p.config.ConfigPath
	if err := p.writeKubeconfig(outputPath, mergeConfig, opts, result); err != nil {
		return nil, err
	}

	result.Metadata["discovered_clusters"] = len(discovered)
	return result, nil
}

func (p *Provider) validateAndPrepare(opts *core.GenerateOptions) error {
	if p.config == nil {
		return errProviderConfigNil
	}

	if opts != nil && opts.Config != nil {
		cfg, ok := opts.Config.(*Config)
		if !ok {
			return errors.New("kubernetes config has unexpected type")
		}
		p.config = cfg
	}

	return p.config.Validate()
}

func (p *Provider) discoverClusters(ctx context.Context) ([]DiscoveredCluster, error) {
	if p.config.MergeOnly {
		return nil, nil
	}

	if p.config.Demo {
		return demoClusters(), nil
	}

	return DiscoverEKSClusters(ctx, p.config, nil)
}

func (p *Provider) buildKubeconfig(discovered []DiscoveredCluster) (*api.Config, []string, error) {
	var discoveredConfig *api.Config
	if len(discovered) > 0 {
		var err error
		discoveredConfig, err = BuildKubeconfig(discovered, p.config.NamingPattern)
		if err != nil {
			return nil, nil, err
		}
	}

	if !p.config.MergeEnabled && !p.config.MergeOnly {
		return discoveredConfig, nil, nil
	}

	outputPath := p.config.ConfigPath
	mergeConfig, mergeFiles, err := MergeKubeconfigs(outputPath, p.config.Merge, discoveredConfig)
	return mergeConfig, mergeFiles, err
}

func (p *Provider) handleDryRun(result *core.Result, discovered []DiscoveredCluster, mergeConfig *api.Config) *core.Result {
	result.Warnings = append(result.Warnings, "dry-run mode: no files were actually created")
	result.Metadata["discovered_clusters"] = len(discovered)
	result.Metadata["dry_run_output"] = p.config.ConfigPath
	if mergeConfig != nil {
		result.Metadata["dry_run_kubeconfig"] = mergeConfig
	}
	return result
}

func (p *Provider) writeKubeconfig(outputPath string, mergeConfig *api.Config, opts *core.GenerateOptions, result *core.Result) error {
	if outputPath == "" {
		return errors.New("kubernetes config path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return fmt.Errorf("create kubeconfig directory: %w", err)
	}

	// Check if output file exists and skip if force is not set
	if _, err := os.Stat(outputPath); err == nil && !opts.Force {
		result.FilesSkipped = append(result.FilesSkipped, outputPath)
		result.Warnings = append(result.Warnings, "kubeconfig already exists, use --force to overwrite")
		return nil
	}

	if err := clientcmd.WriteToFile(*mergeConfig, outputPath); err != nil {
		return fmt.Errorf("write kubeconfig: %w", err)
	}

	result.FilesCreated = append(result.FilesCreated, outputPath)
	return nil
}

// Backup creates a backup of existing configuration files.
func (p *Provider) Backup(_ context.Context) (string, error) {
	if p.config == nil {
		return "", nil
	}
	return core.BackupFile(p.config.ConfigPath)
}

// Restore recovers configuration from a backup.
func (p *Provider) Restore(_ context.Context, _ string) error {
	return errors.New("restore not yet implemented for kubernetes provider")
}

// Clean removes all configuration files generated by this provider.
func (p *Provider) Clean(_ context.Context) error {
	if p.config == nil {
		return errProviderConfigNil
	}

	if p.config.ConfigPath == "" {
		return nil
	}

	path, err := normalizePath(p.config.ConfigPath, errConfigPathEmpty)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove kubeconfig: %w", err)
	}

	return nil
}

func demoClusters() []DiscoveredCluster {
	return []DiscoveredCluster{
		{
			Profile:  "demo",
			Region:   "us-east-1",
			Name:     "demo-app",
			Endpoint: "https://demo-app.example.com",
			CAData:   []byte("demo-ca"),
		},
		{
			Profile:  "demo",
			Region:   "us-west-2",
			Name:     "demo-platform",
			Endpoint: "https://demo-platform.example.com",
			CAData:   []byte("demo-ca"),
		},
	}
}
