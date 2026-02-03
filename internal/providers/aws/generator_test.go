package aws

import "testing"

const testProfilePrefix = "sso_"

func TestBuildConfigContent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfilePrefix = testProfilePrefix
	cfg.SSO.Region = testRegion
	cfg.SSO.RegistrationScopes = defaultSSOScopes
	cfg.SSO.SessionName = defaultSSOSessionName
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

	content, warnings, err := BuildConfigContent(cfg, profiles)
	if err != nil {
		t.Fatalf("BuildConfigContent failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}

	expected := `[sso-session lazycfg]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile sso_prod/Admin]
sso_session = lazycfg
sso_account_id = 111111111111
sso_role_name = Admin
automatically_generated = true

[profile sso_prod/ReadOnly]
sso_session = lazycfg
sso_account_id = 111111111111
sso_role_name = ReadOnly
automatically_generated = true`

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

	content, _, err := BuildConfigContent(cfg, profiles)
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
sso_role_name = AdminAccess
automatically_generated = true`

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

	content, _, err := BuildConfigContent(cfg, profiles)
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
sso_role_name = Admin
automatically_generated = true`

	if content != expected {
		t.Fatalf("config content = %q", content)
	}
}

func TestBuildConfigContentErrorsOnEmptySessionName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL
	cfg.SSO.SessionName = ""

	_, _, err := BuildConfigContent(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty session name")
	}
}

func TestBuildConfigContentErrorsOnEmptyTemplate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfileTemplate = ""
	cfg.SSO.Region = testRegion
	cfg.SSO.StartURL = testStartURL

	_, _, err := BuildConfigContent(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty template")
	}
}

func TestBuildConfigContentRoleChains(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfilePrefix = testProfilePrefix
	cfg.SSO.Region = testRegion
	cfg.SSO.RegistrationScopes = defaultSSOScopes
	cfg.SSO.SessionName = defaultSSOSessionName
	cfg.SSO.StartURL = testStartURL
	cfg.RoleChains = []RoleChain{
		{
			Name:          "prod-readonly",
			RoleARN:       "arn:aws:iam::111111111111:role/ReadOnly",
			SourceProfile: "sso_prod/Admin",
		},
		{
			Name:          "staging-deploy",
			Region:        "us-west-2",
			RoleARN:       "arn:aws:iam::222222222222:role/DeployRole",
			SourceProfile: "sso_staging/PowerUser",
		},
	}

	profiles := []DiscoveredProfile{
		{
			AccountID:   "111111111111",
			AccountName: "prod",
			RoleName:    "Admin",
		},
		{
			AccountID:   "222222222222",
			AccountName: "staging",
			RoleName:    "PowerUser",
		},
	}

	content, warnings, err := BuildConfigContent(cfg, profiles)
	if err != nil {
		t.Fatalf("BuildConfigContent failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}

	expected := `[sso-session lazycfg]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile sso_prod/Admin]
sso_session = lazycfg
sso_account_id = 111111111111
sso_role_name = Admin
automatically_generated = true

[profile sso_staging/PowerUser]
sso_session = lazycfg
sso_account_id = 222222222222
sso_role_name = PowerUser
automatically_generated = true

[profile sso_prod-readonly]
source_profile = sso_prod/Admin
role_arn = arn:aws:iam::111111111111:role/ReadOnly
automatically_generated = true

[profile sso_staging-deploy]
source_profile = sso_staging/PowerUser
role_arn = arn:aws:iam::222222222222:role/DeployRole
region = us-west-2
automatically_generated = true`

	if content != expected {
		t.Fatalf("config content = %q", content)
	}
}

func TestBuildConfigContentRoleChainWarnings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProfilePrefix = testProfilePrefix
	cfg.SSO.Region = testRegion
	cfg.SSO.RegistrationScopes = defaultSSOScopes
	cfg.SSO.SessionName = defaultSSOSessionName
	cfg.SSO.StartURL = testStartURL
	cfg.RoleChains = []RoleChain{
		{
			Name:          "prod-readonly",
			RoleARN:       "arn:aws:iam::111111111111:role/ReadOnly",
			SourceProfile: "sso_prod/Admin",
		},
	}

	profiles := []DiscoveredProfile{}

	_, warnings, err := BuildConfigContent(cfg, profiles)
	if err != nil {
		t.Fatalf("BuildConfigContent failed: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %v", warnings)
	}
}
