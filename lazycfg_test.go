package main

import (
	"testing"

	"github.com/alecthomas/kong"
)

// TestMain ensures that the help and aws subcommands work correctly
func TestMain(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"HelpCommand", []string{"help"}, false},
		{"AWSCommand", []string{"aws"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := CLI{}
			parser := kong.Must(&cli)
			ctx, err := parser.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			err = ctx.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
