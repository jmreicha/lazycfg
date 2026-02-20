// Package cli provides the command-line interface for cfgctl.
package cli

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/jmreicha/cfgctl/internal/core"
	"github.com/jmreicha/cfgctl/internal/providers/aws"
	"github.com/jmreicha/cfgctl/internal/providers/granted"
	"github.com/jmreicha/cfgctl/internal/providers/kubernetes"
	"github.com/jmreicha/cfgctl/internal/providers/ssh"
	"github.com/spf13/cobra"
)

var (
	// Global flags.
	cfgFile       string
	dryRun        bool
	noBackup      bool
	sshConfigPath string
	debug         bool
	verbose       bool

	// Kubernetes generate flags.
	kubeMerge     bool
	kubeMergeOnly bool
	kubeRegions   string
	kubeRoles     string

	// AWS generate flags.
	awsCredentialProcess bool
	awsCredentials       bool
	awsDemo              bool
	awsPrefix            string
	awsPrune             bool
	awsRoleFilters       string
	awsSSOStartURL       string
	awsSSORegion         string
	awsTemplate          string

	// Shared components.
	registry      *core.Registry
	backupManager *core.BackupManager
	config        *core.Config
	engine        *core.Engine
	logger        *slog.Logger
)

// NewRootCmd creates the root command for cfgctl.
func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cfgctl",
		Short: "A tool for creating and managing configurations",

		Version:      version,
		SilenceUsage: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return initializeComponents()
		},
	}

	helpTemplate := strings.ReplaceAll(rootCmd.HelpTemplate(), "Available Commands:", "Commands:")
	usageTemplate := strings.ReplaceAll(rootCmd.UsageTemplate(), "Available Commands:", "Commands:")
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetUsageTemplate(usageTemplate)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: search in standard locations)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate actions without making changes")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose provider output")
	rootCmd.PersistentFlags().BoolVar(&noBackup, "no-backup", false, "skip backup creation before generation")
	rootCmd.PersistentFlags().StringVar(&sshConfigPath, "ssh-config-path", "", "ssh config directory (default: ~/.ssh)")

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
	logLevel := slog.LevelError
	if debug {
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
	if config == nil {
		config = core.NewConfig()
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

	var grantedConfig *granted.Config
	providerConfig = config.GetProviderConfig(granted.ProviderName)
	if providerConfig == nil {
		grantedConfig = granted.DefaultConfig()
	} else {
		typedConfig, ok := providerConfig.(*granted.Config)
		if !ok {
			return fmt.Errorf("granted provider config has unexpected type %T", providerConfig)
		}
		grantedConfig = typedConfig
	}
	if err := registry.Register(granted.NewProvider(grantedConfig)); err != nil {
		return fmt.Errorf("failed to register granted provider: %w", err)
	}

	var awsConfig *aws.Config
	providerConfig = config.GetProviderConfig(aws.ProviderName)
	if providerConfig == nil {
		awsConfig = aws.DefaultConfig()
		config.SetProviderConfig(aws.ProviderName, awsConfig)
	} else {
		typedConfig, ok := providerConfig.(*aws.Config)
		if !ok {
			return fmt.Errorf("aws provider config has unexpected type %T", providerConfig)
		}
		awsConfig = typedConfig
	}
	applyAWSCLIOverrides(awsConfig)
	if err := registry.Register(aws.NewProvider(awsConfig)); err != nil {
		return fmt.Errorf("failed to register aws provider: %w", err)
	}

	var kubernetesConfig *kubernetes.Config
	providerConfig = config.GetProviderConfig(kubernetes.ProviderName)
	if providerConfig == nil {
		kubernetesConfig = kubernetes.DefaultConfig()
		config.SetProviderConfig(kubernetes.ProviderName, kubernetesConfig)
	} else {
		typedConfig, ok := providerConfig.(*kubernetes.Config)
		if !ok {
			return fmt.Errorf("kubernetes provider config has unexpected type %T", providerConfig)
		}
		kubernetesConfig = typedConfig
	}
	applyKubernetesCLIOverrides(kubernetesConfig)
	if err := registry.Register(kubernetes.NewProvider(kubernetesConfig, kubernetes.WithLogger(logger))); err != nil {
		return fmt.Errorf("failed to register kubernetes provider: %w", err)
	}

	return nil
}

func applyKubernetesCLIOverrides(cfg *kubernetes.Config) {
	if cfg == nil {
		return
	}

	if regions := parseCSVFlag(kubeRegions); len(regions) > 0 {
		cfg.AWS.Regions = regions
	}

	if roles := parseCSVFlag(kubeRoles); len(roles) > 0 {
		cfg.AWS.Roles = roles
	}

	if kubeMergeOnly {
		cfg.MergeOnly = true
		cfg.MergeEnabled = true
	} else if kubeMerge {
		cfg.MergeEnabled = true
	}
}

func applyAWSCLIOverrides(cfg *aws.Config) {
	if cfg == nil {
		return
	}

	if awsCredentialProcess {
		cfg.UseCredentialProcess = true
	}

	if awsCredentials {
		cfg.GenerateCredentials = true
	}

	if awsDemo {
		cfg.Demo = true
	}

	if strings.TrimSpace(awsPrefix) != "" {
		cfg.ProfilePrefix = strings.TrimSpace(awsPrefix)
	}

	if awsPrune {
		cfg.Prune = true
	}

	if strings.TrimSpace(awsTemplate) != "" {
		cfg.ProfileTemplate = strings.TrimSpace(awsTemplate)
	}

	if roles := parseCSVFlag(awsRoleFilters); len(roles) > 0 {
		cfg.Roles = roles
	}

	if strings.TrimSpace(awsSSOStartURL) != "" {
		cfg.SSO.StartURL = strings.TrimSpace(awsSSOStartURL)
	}

	if strings.TrimSpace(awsSSORegion) != "" {
		cfg.SSO.Region = strings.TrimSpace(awsSSORegion)
	}
}

func parseCSVFlag(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	set := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}

	if len(set) == 0 {
		return nil
	}

	values := make([]string, 0, len(set))
	for item := range set {
		values = append(values, item)
	}
	sort.Strings(values)
	return values
}
