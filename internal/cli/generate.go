package cli

import (
	"context"
	"fmt"

	"github.com/jmreicha/lazycfg/internal/core"
	"github.com/spf13/cobra"
)

// printGenerateResults outputs the results of generation to stdout.
func printGenerateResults(results map[string]*core.Result) {
	for providerName, result := range results {
		fmt.Printf("\n%s:\n", providerName)

		if len(result.FilesCreated) > 0 {
			fmt.Println("  Files created:")
			for _, file := range result.FilesCreated {
				fmt.Printf("    - %s\n", file)
			}
		}

		if len(result.FilesSkipped) > 0 {
			fmt.Println("  Files skipped:")
			for _, file := range result.FilesSkipped {
				fmt.Printf("    - %s\n", file)
			}
		}

		if result.BackupPath != "" {
			fmt.Printf("  Backup: %s\n", result.BackupPath)
		}

		if len(result.Warnings) > 0 {
			fmt.Println("  Warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("    - %s\n", warning)
			}
		}
	}
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
  lazycfg generate aws kubernetes
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

	return cmd
}
