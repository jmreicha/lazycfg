// Package cli provides the command-line interface for lazycfg.
package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jmreicha/lazycfg/internal/core"
	"github.com/jmreicha/lazycfg/internal/providers/ssh"
	"github.com/spf13/cobra"
)

var (
	// Global flags.
	cfgFile       string
	dryRun        bool
	noBackup      bool
	sshConfigPath string
	verbose       bool

	// Shared components.
	registry      *core.Registry
	backupManager *core.BackupManager
	config        *core.Config
	engine        *core.Engine
	logger        *slog.Logger
)

// NewRootCmd creates the root command for lazycfg.
func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "lazycfg",
		Short: "A tool for creating and managing configurations",
		Long: `lazycfg simplifies the creation and management of complicated configurations.
	It provides a plugin-based architecture for managing AWS, Kubernetes, SSH, and other configurations.

	Use --ssh-config-path to override the default SSH config directory.`,

		Version:      version,
		SilenceUsage: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return initializeComponents()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: search in standard locations)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate actions without making changes")
	rootCmd.PersistentFlags().BoolVar(&noBackup, "no-backup", false, "skip backup creation before generation")
	rootCmd.PersistentFlags().StringVar(&sshConfigPath, "ssh-config-path", "", "ssh config directory (default: ~/.ssh)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newCleanCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newVersionCmd(version))

	return rootCmd
}

// initializeComponents sets up the core components needed by all commands.
func initializeComponents() error {
	// Set up logger
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Load configuration
	var err error
	config, err = core.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with CLI flags
	if verbose {
		config.Verbose = true
	}
	if dryRun {
		config.DryRun = true
	}
	if noBackup {
		config.NoBackup = true
	}

	// Initialize core components
	registry = core.NewRegistry()
	backupManager = core.NewBackupManager("")
	engine = core.NewEngine(registry, backupManager, config, logger)

	// Register providers
	var sshConfig *ssh.Config
	providerConfig := config.GetProviderConfig(ssh.ProviderName)
	if providerConfig == nil {
		sshConfig = ssh.DefaultConfig()
	} else {
		typedConfig, ok := providerConfig.(*ssh.Config)
		if !ok {
			return fmt.Errorf("ssh provider config has unexpected type %T", providerConfig)
		}
		sshConfig = typedConfig
	}
	if sshConfigPath != "" {
		sshConfig.ConfigPath = sshConfigPath
	}
	if err := registry.Register(ssh.NewProvider(sshConfig)); err != nil {
		return fmt.Errorf("failed to register ssh provider: %w", err)
	}

	return nil
}
