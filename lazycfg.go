package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

// CLI represents the command-line interface structure
type CLI struct {
	Help HelpCmd `cmd:"" default:"1" help:"Show help."`
	AWS  AWSCmd  `cmd:"" help:"Configure AWS settings."`
}

// HelpCmd represents the help command
type HelpCmd struct{}

// Run executes the help command
func (h *HelpCmd) Run() error {
	fmt.Println("Showing help...")
	return nil
}

// AWSCmd represents the AWS configuration command
type AWSCmd struct{}

// Run executes the AWS configuration command
func (a *AWSCmd) Run() error {
	fmt.Println("Configuring AWS settings...")
	// Stub for configuring .aws/config and .aws/credentials
	return nil
}

func main() {
	cli := CLI{}

	ctx := kong.Parse(&cli)

	err := ctx.Run()

	ctx.FatalIfErrorf(err)
}
