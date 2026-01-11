package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"github.com/jmreicha/lazycfg/internal/cmd/configure"
	"github.com/jmreicha/lazycfg/internal/cmd/generate"
)

// Build information set via ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// CLI represents the command-line interface structure
type CLI struct {
	// Flags
	Debug   bool             `short:"d" help:"Emit debug logs in addition to info logs."`
	Version kong.VersionFlag `short:"v" help:"Show version information."`

	Configure ConfigureCmd `cmd:"" help:"Guided configuration setup."`
	Generate  struct {
		Granted   GrantedCmd   `cmd:"" help:"Generate Granted configuration."`
		Steampipe SteampipeCmd `cmd:"" help:"Generate Steampipe configuration."`
	} `cmd:"" help:"Generate config for a specific tool."`
	Clean CleanCmd `cmd:"" help:"Clean up configuration files."`
}

// ConfigureCmd represents the configure command
type ConfigureCmd struct{}

// Run executes the configure command
func (c *ConfigureCmd) Run() error {
	color.Blue("Running configuration...")
	return configure.RunConfiguration()
}

// CleanCmd represents the granted subcommand
type CleanCmd struct{}

// Run executes the clean subcommand
func (g *CleanCmd) Run() error {
	color.Blue("Cleaning configurations...")
	return nil
}

// GrantedCmd represents the granted subcommand
type GrantedCmd struct{}

// Run executes the granted subcommand
func (g *GrantedCmd) Run() error {
	color.Blue("Generating Granted configuration...")
	return generate.CreateGrantedConfiguration(generate.GrantedConfigPath)
}

// SteampipeCmd represents the steampipe subcommand
type SteampipeCmd struct{}

// SteampipeCmd executes the steampipe subcommand
func (s *SteampipeCmd) Run() error {
	color.Blue("Generating Steampipe configuration...")
	return generate.CreateSteampipeConfiguration(generate.SteamipeConfigPath)
}

func main() {
	cli := CLI{}

	// Display help if no args are provided instead of an error message
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "--help")
	}

	ctx := kong.Parse(&cli,
		kong.Name("lazycfg"),
		kong.Description("A tool for creating and managing configurations."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:   true,
			FlagsLast: true,
		}),
		kong.Vars{
			"version": version,
		},
	)

	err := ctx.Run()

	ctx.FatalIfErrorf(err)
}
