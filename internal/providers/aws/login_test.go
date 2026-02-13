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
	cfg.SSO.StartURL = testStartURL
	cfg.SSO.Region = testRegion

	if err := ensureSSOSessionBlock(cfg); err != nil {
		t.Fatalf("ensureSSOSessionBlock failed: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[sso-session cfgctl]") {
		t.Fatal("missing sso-session header")
	}
	if !strings.Contains(content, "sso_start_url = "+testStartURL) {
		t.Fatal("missing sso_start_url")
	}
	if !strings.Contains(content, "sso_region = "+testRegion) {
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
	cfg.SSO.StartURL = testStartURL
	cfg.SSO.Region = testRegion

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
	if !strings.Contains(content, "[sso-session cfgctl]") {
		t.Fatal("missing sso-session header")
	}
}

func TestEnsureSSOSessionBlockSkipsIfPresent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")

	existing := "[sso-session cfgctl]\nsso_start_url = " + testStartURL + "\nsso_region = " + testRegion + "\n"
	if err := os.WriteFile(cfgPath, []byte(existing), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := DefaultConfig()
	cfg.ConfigPath = cfgPath
	cfg.SSO.StartURL = testStartURL
	cfg.SSO.Region = testRegion

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
	cfg.SSO.StartURL = testStartURL
	cfg.SSO.Region = testRegion

	err := runSSOLogin(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "aws sso login") {
		t.Fatalf("unexpected error: %v", err)
	}
}
