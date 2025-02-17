package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/jmreicha/lazycfg/internal/cmd/configure"
	"github.com/jmreicha/lazycfg/internal/cmd/generate"
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
}

// ConfigureCmd represents the configure command
type ConfigureCmd struct{}

// Run executes the configure command
func (c *ConfigureCmd) Run() error {
	fmt.Println("Running configuration...")
	return configure.RunConfiguration()
}

// GrantedCmd represents the granted subcommand
type GrantedCmd struct{}

// Run executes the granted subcommand
func (g *GrantedCmd) Run() error {
	fmt.Println("Generating Granted configuration...")
	return generate.CreateGrantedConfiguration()
}

// SteampipeCmd represents the steampipe subcommand
type SteampipeCmd struct{}

// SteampipeCmd executes the steampipe subcommand
func (s *SteampipeCmd) Run() error {
	fmt.Println("Generating Steampipe configuration...")
	return generate.CreateSteampipeConfiguration()
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
			// TODO(jmreicha): See if there is a way to dynamically set this
			"version": "v1.0.0",
		},
	)

	err := ctx.Run()

	ctx.FatalIfErrorf(err)
}
