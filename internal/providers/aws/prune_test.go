package aws

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeConfigContentPrunesMarkedProfiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	initial := `[sso-session cfgctl]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile keep-unmarked]
sso_session = cfgctl
sso_account_id = 111111111111
sso_role_name = Admin

[profile stale-marked]
sso_session = cfgctl
sso_account_id = 222222222222
sso_role_name = ReadOnly
sso_auto_populated = true
`

	if err := os.WriteFile(configPath, []byte(initial), 0600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	generated := `[sso-session cfgctl]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile fresh-marked]
sso_session = cfgctl
sso_account_id = 333333333333
sso_role_name = Admin
sso_auto_populated = true
`

	merged, err := mergeConfigContent(configPath, generated, []string{"fresh-marked"}, "sso_auto_populated", "cfgctl")
	if err != nil {
		t.Fatalf("merge config content: %v", err)
	}

	expected := `[profile keep-unmarked]
sso_session = cfgctl
sso_account_id = 111111111111
sso_role_name = Admin

[sso-session cfgctl]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile fresh-marked]
sso_session = cfgctl
sso_account_id = 333333333333
sso_role_name = Admin
sso_auto_populated = true`

	if merged != expected {
		t.Fatalf("merged content = %q", merged)
	}
}

func TestMergeConfigContentSkipsPruneWhenMarkerEmpty(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	initial := `[profile unmarked]
sso_session = cfgctl
sso_account_id = 111111111111
sso_role_name = Admin
`

	if err := os.WriteFile(configPath, []byte(initial), 0600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	generated := `[sso-session cfgctl]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
`

	merged, err := mergeConfigContent(configPath, generated, []string{}, "", "cfgctl")
	if err != nil {
		t.Fatalf("merge config content: %v", err)
	}

	if merged != generated {
		t.Fatalf("merged content = %q", merged)
	}
}
