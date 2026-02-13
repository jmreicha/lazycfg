package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jmreicha/lazycfg/internal/core"
	"github.com/spf13/cobra"
)

// printGenerateResults outputs the results of generation to stdout.
func printGenerateResults(results map[string]*core.Result) {
	for providerName, result := range results {
		fmt.Printf("\n%s:\n", formatSection(providerName))

		if len(result.FilesCreated) > 0 {
			fmt.Println(formatLabel("  Files created:"))
			for _, file := range result.FilesCreated {
				fmt.Printf("    - %s\n", formatPath(file))
			}
		}

		if len(result.FilesSkipped) > 0 {
			fmt.Println(formatLabel("  Files skipped:"))
			for _, file := range result.FilesSkipped {
				fmt.Printf("    - %s\n", formatPath(file))
			}
		}

		if result.BackupPath != "" {
			fmt.Printf("%s %s\n", formatLabel("  Backup:"), formatPath(result.BackupPath))
		}

		if len(result.Warnings) > 0 {
			fmt.Println(formatLabel("  Warnings:"))
			for _, warning := range result.Warnings {
				fmt.Printf("    - %s\n", formatWarning(warning))
			}
		}
	}
}

func formatLabel(value string) string {
	return colorize(value, "1")
}

func formatPath(value string) string {
	return colorize(value, "36")
}

func formatSection(value string) string {
	return colorize(value, "1;36")
}

func formatWarning(value string) string {
	return colorize(value, "33")
}

func colorize(value, code string) string {
	if !supportsColor() {
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
  lazycfg generate aws
  lazycfg generate kubernetes
  lazycfg generate ssh --ssh-config-path ~/.ssh
  lazycfg generate aws ssh
  lazycfg generate all
  lazycfg generate --dry-run
  lazycfg generate --force`,
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
	cmd.Flags().BoolVar(&kubeDemo, "kube-demo", false, "use fake kubernetes discovery data")
	cmd.Flags().BoolVar(&kubeMerge, "kube-merge", false, "merge existing kubeconfig files")
	cmd.Flags().BoolVar(&kubeMergeOnly, "kube-merge-only", false, "merge existing kubeconfig files without AWS discovery")
	cmd.Flags().StringVar(&kubeProfiles, "kube-profiles", "", "comma-separated AWS profile names")
	cmd.Flags().StringVar(&kubeRegions, "kube-regions", "", "comma-separated AWS regions")

	return cmd
}
