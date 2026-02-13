package aws

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// awsCommand is the AWS CLI binary name. Overridden in tests.
var awsCommand = "aws"

// runSSOLogin ensures the sso-session block exists in the AWS config file
// and runs `aws sso login` interactively so the user can authenticate.
func runSSOLogin(ctx context.Context, cfg *Config) error {
	if err := ensureSSOSessionBlock(cfg); err != nil {
		return fmt.Errorf("ensure sso-session block: %w", err)
	}

	// #nosec G204 -- session name is from user configuration, not external input.
	cmd := exec.CommandContext(ctx, awsCommand, "sso", "login", "--sso-session", cfg.SSO.SessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws sso login: %w", err)
	}

	return nil
}

// ensureSSOSessionBlock appends the [sso-session] block to the AWS config
// file if it does not already exist.
func ensureSSOSessionBlock(cfg *Config) error {
	path, err := normalizeConfigPath(cfg.ConfigPath)
	if err != nil {
		return err
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read config %q: %w", path, err)
	}

	header := fmt.Sprintf("[sso-session %s]", cfg.SSO.SessionName)
	if strings.Contains(string(existing), header) {
		return nil
	}

	var block strings.Builder
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		block.WriteString("\n")
	}
	block.WriteString(header + "\n")
	block.WriteString("sso_start_url = " + cfg.SSO.StartURL + "\n")
	block.WriteString("sso_region = " + cfg.SSO.Region + "\n")
	block.WriteString("sso_registration_scopes = " + cfg.SSO.RegistrationScopes + "\n")

	// #nosec G306 -- AWS config files use 0600 permissions.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open config %q for append: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(block.String()); err != nil {
		return fmt.Errorf("append sso-session block to %q: %w", path, err)
	}

	return nil
}
