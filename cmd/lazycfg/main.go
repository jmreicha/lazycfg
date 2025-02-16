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
	Configure ConfigureCmd `cmd:"" help:"Guided configuration setup."`
	Generate  GenerateCmd  `cmd:"" help:"Run configuration setup for a specific tool."`
}

// ConfigureCmd represents the configure command
type ConfigureCmd struct{}

// Run executes the configure command
func (c *ConfigureCmd) Run() error {
	fmt.Println("Running configuration...")
	return configure.RunConfiguration()
}

// GenerateCmd represents the generate configuration command
type GenerateCmd struct{}

// Run executes the tool configuration command
func (t *GenerateCmd) Run() error {
	fmt.Println("Running configuration generation...")
	return nil
}

// GrantedCmd represents the granted subcommand
type GrantedCmd struct{}

// Run executes the granted subcommand
func (g *GrantedCmd) Run() error {
	fmt.Println("Running configuration generation for Granted...")
	return generate.CreateGrantedConfiguration()
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
	)

	err := ctx.Run()

	ctx.FatalIfErrorf(err)
}
