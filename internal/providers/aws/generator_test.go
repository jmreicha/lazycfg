package aws

import "testing"

func TestBuildConfigContent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfilePrefix = "sso_"
	cfg.SSO.Region = testRegion
	cfg.SSO.RegistrationScopes = "sso:account:access"
	cfg.SSO.SessionName = "lazycfg"
	cfg.SSO.StartURL = testStartURL

	profiles := []DiscoveredProfile{
		{
			AccountID:   "111111111111",
			AccountName: "prod",
			RoleName:    "Admin",
		},
		{
			AccountID:   "111111111111",
			AccountName: "prod",
			RoleName:    "ReadOnly",
		},
	}

	content, err := BuildConfigContent(cfg, profiles)
	if err != nil {
		t.Fatalf("BuildConfigContent failed: %v", err)
	}

	expected := `[sso-session lazycfg]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile sso_prod/Admin]
sso_session = lazycfg
sso_account_id = 111111111111
sso_role_name = Admin

[profile sso_prod/ReadOnly]
sso_session = lazycfg
sso_account_id = 111111111111
sso_role_name = ReadOnly`

	if content != expected {
		t.Fatalf("config content = %q", content)
	}
}

func TestBuildConfigContentTemplateAliases(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfileTemplate = "{{ .account }}-{{ .role }}"
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL

	profiles := []DiscoveredProfile{
		{
			AccountID:   "123456789012",
			AccountName: "prod-account",
			RoleName:    "AdminAccess",
		},
	}

	content, err := BuildConfigContent(cfg, profiles)
	if err != nil {
		t.Fatalf("BuildConfigContent failed: %v", err)
	}

	expected := `[sso-session lazycfg]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile prod-account-AdminAccess]
sso_session = lazycfg
sso_account_id = 123456789012
sso_role_name = AdminAccess`

	if content != expected {
		t.Fatalf("config content = %q", content)
	}
}

func TestBuildConfigContentOverwritesOnCollision(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfileTemplate = "{{ .AccountName }}-{{ .RoleName }}"
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL

	profiles := []DiscoveredProfile{
		{
			AccountID:   "111111111111",
			AccountName: "prod",
			RoleName:    "Admin",
		},
		{
			AccountID:   "999999999999",
			AccountName: "prod",
			RoleName:    "Admin",
		},
	}

	content, err := BuildConfigContent(cfg, profiles)
	if err != nil {
		t.Fatalf("BuildConfigContent failed: %v", err)
	}

	expected := `[sso-session lazycfg]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile prod-Admin]
sso_session = lazycfg
sso_account_id = 999999999999
sso_role_name = Admin`

	if content != expected {
		t.Fatalf("config content = %q", content)
	}
}

func TestBuildConfigContentErrorsOnEmptySessionName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL
	cfg.SSO.SessionName = ""

	_, err := BuildConfigContent(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty session name")
	}
}

func TestBuildConfigContentErrorsOnEmptyTemplate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfileTemplate = ""
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL

	_, err := BuildConfigContent(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty template")
	}
}
