package ssh_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmreicha/lazycfg/internal/core"
	"github.com/jmreicha/lazycfg/internal/providers/ssh"
)

// TestE2E_SSHProvider_GenerateWorkflow tests the complete 'lazycfg generate ssh' workflow.
func TestE2E_SSHProvider_GenerateWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Setup: Create a configuration
	config := &ssh.Config{
		Enabled:    true,
		ConfigPath: tmpDir,
		GlobalOptions: map[string]string{
			"ServerAliveInterval": "60",
			"ServerAliveCountMax": "3",
		},
		Hosts: []ssh.HostConfig{
			{
				Host:         "prod1.example.com",
				Hostname:     "192.168.1.10",
				User:         "admin",
				Port:         22,
				IdentityFile: "/home/user/.ssh/prod_key",
			},
			{
				Host:         "dev*.example.com",
				Hostname:     "192.168.2.10",
				User:         "developer",
				Port:         2222,
				IdentityFile: "/home/user/.ssh/dev_key",
			},
		},
	}

	// Initialize provider
	provider := ssh.NewProvider(config)

	ctx := context.Background()

	// Step 1: Validate provider
	t.Log("Step 1: Validating provider")
	err := provider.Validate(ctx)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Step 2: Generate configuration (first time)
	t.Log("Step 2: Generating configuration (first time)")
	opts := &core.GenerateOptions{
		DryRun: false,
		Force:  false,
	}

	result, err := provider.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Verify results
	if result.Provider != "ssh" {
		t.Errorf("provider name = %s, want ssh", result.Provider)
	}

	if len(result.FilesCreated) != 1 {
		t.Errorf("files created = %d, want 1", len(result.FilesCreated))
	}

	if result.FilesCreated[0] != configPath {
		t.Errorf("config path = %s, want %s", result.FilesCreated[0], configPath)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Verify file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Step 3: Parse and verify configuration content
	t.Log("Step 3: Verifying configuration content")
	cfg, err := ssh.ParseConfig(configPath)
	if err != nil {
		t.Fatalf("failed to parse generated config: %v", err)
	}

	// Verify hosts
	prodHost := ssh.FindHost(cfg, "prod1.example.com")
	if prodHost == nil {
		t.Fatal("prod1.example.com not found")
	}

	devHost := ssh.FindHost(cfg, "dev1.example.com")
	if devHost == nil {
		t.Fatal("dev*.example.com pattern not found")
	}

	// Verify global options
	interval, err := cfg.Get("anyhost.com", "ServerAliveInterval")
	if err != nil {
		t.Errorf("failed to get ServerAliveInterval: %v", err)
	}
	if interval != "60" {
		t.Errorf("ServerAliveInterval = %s, want 60", interval)
	}

	// Step 4: Attempt to regenerate without force (should skip)
	t.Log("Step 4: Attempting to regenerate without force")
	result, err = provider.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("second generation failed: %v", err)
	}

	if len(result.FilesSkipped) != 1 {
		t.Errorf("files skipped = %d, want 1", len(result.FilesSkipped))
	}

	// Step 5: Regenerate with force
	t.Log("Step 5: Regenerating with force")
	opts.Force = true
	result, err = provider.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("forced generation failed: %v", err)
	}

	if len(result.FilesCreated) != 1 {
		t.Errorf("files created with force = %d, want 1", len(result.FilesCreated))
	}

	// Step 6: Test dry-run mode
	t.Log("Step 6: Testing dry-run mode")
	dryRunOpts := &core.GenerateOptions{
		DryRun: true,
		Force:  false,
	}

	result, err = provider.Generate(ctx, dryRunOpts)
	if err != nil {
		t.Fatalf("dry-run generation failed: %v", err)
	}

	foundDryRunWarning := false
	for _, warning := range result.Warnings {
		if contains(warning, "dry-run") {
			foundDryRunWarning = true
			break
		}
	}
	if !foundDryRunWarning {
		t.Error("expected dry-run warning")
	}

	// Step 7: Clean up
	t.Log("Step 7: Cleaning up")
	err = provider.Clean(ctx)
	if err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}

// TestE2E_SSHProvider_UpdateExistingConfig tests updating an existing SSH config.
func TestE2E_SSHProvider_UpdateExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Step 1: Create existing SSH config manually
	t.Log("Step 1: Creating existing SSH config")
	existingConfig := `# Existing SSH Configuration

Host existing1.com
    HostName existing1.example.com
    User existing-user
    Port 22

Host existing2.com
    HostName existing2.example.com
    User another-user
    Port 2222

Host *
    ServerAliveInterval 30
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0600); err != nil {
		t.Fatalf("failed to create existing config: %v", err)
	}

	// Step 2: Initialize provider with new hosts
	t.Log("Step 2: Initializing provider with new configuration")
	config := &ssh.Config{
		Enabled:    true,
		ConfigPath: tmpDir,
		GlobalOptions: map[string]string{
			"ServerAliveInterval":   "60", // Update existing
			"StrictHostKeyChecking": "no", // New option
		},
		Hosts: []ssh.HostConfig{
			{
				Host:     "new1.com",
				Hostname: "new1.example.com",
				User:     "newuser",
				Port:     22,
			},
		},
	}

	provider := ssh.NewProvider(config)
	ctx := context.Background()

	// Step 3: Generate with force to update
	t.Log("Step 3: Generating with force to update existing config")
	opts := &core.GenerateOptions{
		DryRun: false,
		Force:  true,
	}

	result, err := provider.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(result.FilesCreated) != 1 {
		t.Errorf("files created = %d, want 1", len(result.FilesCreated))
	}

	// Step 4: Verify merged configuration
	t.Log("Step 4: Verifying merged configuration")
	cfg, err := ssh.ParseConfig(configPath)
	if err != nil {
		t.Fatalf("failed to parse updated config: %v", err)
	}

	// Verify old hosts still exist
	if host := ssh.FindHost(cfg, "existing1.com"); host == nil {
		t.Error("existing1.com was lost during update")
	}
	if host := ssh.FindHost(cfg, "existing2.com"); host == nil {
		t.Error("existing2.com was lost during update")
	}

	// Verify new host was added
	if host := ssh.FindHost(cfg, "new1.com"); host == nil {
		t.Error("new1.com was not added")
	}

	// Verify global option was updated
	interval, err := cfg.Get("anyhost.com", "ServerAliveInterval")
	if err != nil {
		t.Errorf("failed to get ServerAliveInterval: %v", err)
	}
	if interval != "60" {
		t.Errorf("ServerAliveInterval = %s, want 60 (should be updated)", interval)
	}

	// Verify new global option was added
	strictCheck, err := cfg.Get("anyhost.com", "StrictHostKeyChecking")
	if err != nil {
		t.Errorf("failed to get StrictHostKeyChecking: %v", err)
	}
	if strictCheck != "no" {
		t.Errorf("StrictHostKeyChecking = %s, want no", strictCheck)
	}
}

// TestE2E_SSHProvider_MissingConfig tests behavior when config file is missing.
func TestE2E_SSHProvider_MissingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ssh.Config{
		Enabled:    true,
		ConfigPath: tmpDir,
		Hosts: []ssh.HostConfig{
			{
				Host:     "test.com",
				Hostname: "test.example.com",
				User:     "testuser",
				Port:     22,
			},
		},
	}

	provider := ssh.NewProvider(config)
	ctx := context.Background()

	opts := &core.GenerateOptions{
		DryRun: false,
		Force:  false,
	}

	// Generate should succeed and create new file
	result, err := provider.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(result.FilesCreated) != 1 {
		t.Errorf("expected 1 file created, got %d", len(result.FilesCreated))
	}
}

// TestE2E_SSHProvider_EmptyConfig tests behavior with empty config directory.
func TestE2E_SSHProvider_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ssh.Config{
		Enabled:       true,
		ConfigPath:    tmpDir,
		GlobalOptions: map[string]string{},
		Hosts:         []ssh.HostConfig{},
	}

	provider := ssh.NewProvider(config)
	ctx := context.Background()

	opts := &core.GenerateOptions{
		DryRun: false,
		Force:  false,
	}

	result, err := provider.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Should not create file if no hosts configured
	if len(result.FilesCreated) != 0 {
		t.Errorf("expected no files created, got %d", len(result.FilesCreated))
	}

	// Should have warning
	foundWarning := false
	for _, warning := range result.Warnings {
		if contains(warning, "no hosts") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected 'no hosts' warning")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
