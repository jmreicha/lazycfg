package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jmreicha/cfgctl/internal/core"
	"github.com/spf13/cobra"
)

// printGenerateResults outputs the results of generation to stdout.
func printGenerateResults(results map[string]*core.Result) {
	colorEnabled := supportsColor()
	for providerName, result := range results {
		fmt.Printf("\n%s:\n", formatSection(colorEnabled, providerName))

		printMetadataSummary(colorEnabled, result.Metadata)

		if len(result.FilesCreated) > 0 {
			fmt.Println(formatLabel(colorEnabled, "  Files created:"))
			for _, file := range result.FilesCreated {
				fmt.Printf("    - %s\n", formatPath(colorEnabled, file))
			}
		}

		if len(result.FilesSkipped) > 0 {
			fmt.Println(formatLabel(colorEnabled, "  Files skipped:"))
			for _, file := range result.FilesSkipped {
				fmt.Printf("    - %s\n", formatPath(colorEnabled, file))
			}
		}

		if result.BackupPath != "" {
			fmt.Printf("%s %s\n", formatLabel(colorEnabled, "  Backup:"), formatPath(colorEnabled, result.BackupPath))
		}

		if len(result.Warnings) > 0 {
			fmt.Println(formatLabel(colorEnabled, "  Warnings:"))
			for _, warning := range result.Warnings {
				fmt.Printf("    - %s\n", formatWarning(colorEnabled, warning))
			}
		}
	}
}

func printMetadataSummary(colorEnabled bool, metadata map[string]interface{}) {
	if len(metadata) == 0 {
		return
	}

	if clusters, ok := metadata["discovered_clusters"]; ok {
		fmt.Printf("  %s %v\n", formatLabel(colorEnabled, "Clusters discovered:"), clusters)
	}
	if regions, ok := metadata["regions"]; ok {
		if rs, ok := regions.([]string); ok {
			fmt.Printf("  %s %s\n", formatLabel(colorEnabled, "Regions:"), strings.Join(rs, ", "))
		}
	}
	if authMode, ok := metadata["auth_mode"]; ok {
		fmt.Printf("  %s %v\n", formatLabel(colorEnabled, "Auth:"), authMode)
	}
	if mergeFiles, ok := metadata["merge_files"]; ok {
		if files, ok := mergeFiles.([]string); ok && len(files) > 0 {
			fmt.Printf("  %s %d files\n", formatLabel(colorEnabled, "Merged:"), len(files))
		}
	}
}

func formatLabel(colorEnabled bool, value string) string {
	return colorize(colorEnabled, value, "1")
}

func formatPath(colorEnabled bool, value string) string {
	return colorize(colorEnabled, value, "36")
}

func formatSection(colorEnabled bool, value string) string {
	return colorize(colorEnabled, value, "1;36")
}

func formatWarning(colorEnabled bool, value string) string {
	return colorize(colorEnabled, value, "33")
}

func colorize(colorEnabled bool, value, code string) string {
	if !colorEnabled {
		return value
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", code, value)
}

func supportsColor() bool {
	if !isTerminal(os.Stdout) {
		return false
	}
	if noColor := os.Getenv("NO_COLOR"); noColor != "" {
		return false
	}
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}
	return true
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func newGenerateCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "generate [provider...]",
		Short: "Generate configuration files",
		Long: `Generate configuration files for one or more providers.
If no providers are specified, all registered providers will be executed.

Examples:
  cfgctl generate aws
  cfgctl generate kubernetes
  cfgctl generate ssh --ssh-config-path ~/.ssh
  cfgctl generate aws ssh
  cfgctl generate all
  cfgctl generate --dry-run
  cfgctl generate --force`,
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()

			// Handle "all" as an explicit provider name
			providers := args
			if len(args) == 1 && args[0] == "all" {
				providers = nil // Empty list means "all providers"
			}

			opts := &core.ExecuteOptions{
				Providers: providers,
				DryRun:    dryRun,
				Force:     force,
				NoBackup:  noBackup,
				Verbose:   verbose,
			}

			results, err := engine.Execute(ctx, opts)
			if err != nil {
				return err
			}

			printGenerateResults(results)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")
	cmd.Flags().BoolVar(&awsCredentialProcess, "aws-credential-process", false, "use credential_process for AWS profiles")
	cmd.Flags().BoolVar(&awsCredentials, "aws-credentials", false, "generate AWS credentials output")
	cmd.Flags().BoolVar(&awsDemo, "aws-demo", false, "use fake AWS discovery data")
	cmd.Flags().StringVar(&awsPrefix, "aws-prefix", "", "prefix for generated AWS profile names")
	cmd.Flags().BoolVar(&awsPrune, "aws-prune", false, "remove stale AWS profiles with marker key")
	cmd.Flags().StringVar(&awsRoleFilters, "aws-roles", "", "comma-separated AWS role names")
	cmd.Flags().StringVar(&awsSSOStartURL, "aws-sso-url", "", "AWS SSO start URL")
	cmd.Flags().StringVar(&awsSSORegion, "aws-sso-region", "", "AWS SSO region")
	cmd.Flags().StringVar(&awsTemplate, "aws-template", "", "template for AWS profile names")
	cmd.Flags().BoolVar(&kubeMerge, "kube-merge", false, "merge existing kubeconfig files")
	cmd.Flags().BoolVar(&kubeMergeOnly, "kube-merge-only", false, "merge existing kubeconfig files without AWS discovery")
	cmd.Flags().StringVar(&kubeRegions, "kube-regions", "", "comma-separated AWS regions")
	cmd.Flags().StringVar(&kubeRoles, "kube-roles", "", "comma-separated role names to filter profiles (e.g. adminaccess)")
	cmd.Flags().BoolVar(&steampipeIgnoreErrors, "steampipe-ignore-errors", false, "add ignore_error_codes (AccessDenied, UnauthorizedOperation) to all connections")
	cmd.Flags().StringVar(&steampipeRegions, "steampipe-regions", "", "comma-separated AWS regions for steampipe connections")

	return cmd
}
