package aws

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSSOSessionBlockCreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")

	cfg := DefaultConfig()
	cfg.ConfigPath = cfgPath
	cfg.SSO.StartURL = "https://example.awsapps.com/start"
	cfg.SSO.Region = "us-east-1"

	if err := ensureSSOSessionBlock(cfg); err != nil {
		t.Fatalf("ensureSSOSessionBlock failed: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[sso-session lazycfg]") {
		t.Fatal("missing sso-session header")
	}
	if !strings.Contains(content, "sso_start_url = https://example.awsapps.com/start") {
		t.Fatal("missing sso_start_url")
	}
	if !strings.Contains(content, "sso_region = us-east-1") {
		t.Fatal("missing sso_region")
	}
}

func TestEnsureSSOSessionBlockAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")

	existing := "[profile existing]\nregion = us-west-2\n"
	if err := os.WriteFile(cfgPath, []byte(existing), 0600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	cfg := DefaultConfig()
	cfg.ConfigPath = cfgPath
	cfg.SSO.StartURL = "https://example.awsapps.com/start"
	cfg.SSO.Region = "us-east-1"

	if err := ensureSSOSessionBlock(cfg); err != nil {
		t.Fatalf("ensureSSOSessionBlock failed: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[profile existing]") {
		t.Fatal("existing content was lost")
	}
	if !strings.Contains(content, "[sso-session lazycfg]") {
		t.Fatal("missing sso-session header")
	}
}

func TestEnsureSSOSessionBlockSkipsIfPresent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")

	existing := "[sso-session lazycfg]\nsso_start_url = https://example.awsapps.com/start\nsso_region = us-east-1\n"
	if err := os.WriteFile(cfgPath, []byte(existing), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := DefaultConfig()
	cfg.ConfigPath = cfgPath
	cfg.SSO.StartURL = "https://example.awsapps.com/start"
	cfg.SSO.Region = "us-east-1"

	if err := ensureSSOSessionBlock(cfg); err != nil {
		t.Fatalf("ensureSSOSessionBlock failed: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if string(data) != existing {
		t.Fatalf("file was modified when block already existed:\n%s", string(data))
	}
}

func TestRunSSOLoginMissingBinary(t *testing.T) {
	orig := awsCommand
	awsCommand = "nonexistent-aws-cli-binary"
	t.Cleanup(func() { awsCommand = orig })

	cfg := DefaultConfig()
	cfg.ConfigPath = filepath.Join(t.TempDir(), "config")
	cfg.SSO.StartURL = "https://example.awsapps.com/start"
	cfg.SSO.Region = "us-east-1"

	err := runSSOLogin(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "aws sso login") {
		t.Fatalf("unexpected error: %v", err)
	}
}
