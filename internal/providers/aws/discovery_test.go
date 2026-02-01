package aws

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

type mockSSOClient struct {
	accountsPages []*sso.ListAccountsOutput
	rolesPages    map[string][]*sso.ListAccountRolesOutput
	listErr       error
	rolesErr      error
}

func (m *mockSSOClient) ListAccounts(_ context.Context, _ *sso.ListAccountsInput, _ ...func(*sso.Options)) (*sso.ListAccountsOutput, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if len(m.accountsPages) == 0 {
		return &sso.ListAccountsOutput{}, nil
	}
	output := m.accountsPages[0]
	m.accountsPages = m.accountsPages[1:]
	return output, nil
}

func (m *mockSSOClient) ListAccountRoles(_ context.Context, params *sso.ListAccountRolesInput, _ ...func(*sso.Options)) (*sso.ListAccountRolesOutput, error) {
	if m.rolesErr != nil {
		return nil, m.rolesErr
	}
	if params == nil || params.AccountId == nil {
		return nil, errors.New("account id missing")
	}
	pages := m.rolesPages[aws.ToString(params.AccountId)]
	if len(pages) == 0 {
		return &sso.ListAccountRolesOutput{}, nil
	}
	output := pages[0]
	m.rolesPages[aws.ToString(params.AccountId)] = pages[1:]
	return output, nil
}

func TestDiscoverProfiles(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SSO.Region = "us-east-1"
	cfg.SSO.StartURL = "https://example.awsapps.com/start"
	cfg.TokenCachePaths = []string{"/cache"}
	cfg.Roles = []string{"Admin", "ReadOnly"}

	factory := func(_ context.Context, _, _ string) (SSOClient, error) {
		return &mockSSOClient{
			accountsPages: []*sso.ListAccountsOutput{
				{
					AccountList: []types.AccountInfo{
						{
							AccountId:   aws.String("111111111111"),
							AccountName: aws.String("prod"),
						},
					},
					NextToken: aws.String("next"),
				},
				{
					AccountList: []types.AccountInfo{
						{
							AccountId:   aws.String("222222222222"),
							AccountName: aws.String("staging"),
						},
					},
				},
			},
			rolesPages: map[string][]*sso.ListAccountRolesOutput{
				"111111111111": {
					{
						RoleList:  []types.RoleInfo{{RoleName: aws.String("Admin")}, {RoleName: aws.String("Audit")}},
						NextToken: aws.String("more"),
					},
					{
						RoleList: []types.RoleInfo{{RoleName: aws.String("ReadOnly")}},
					},
				},
				"222222222222": {
					{
						RoleList: []types.RoleInfo{{RoleName: aws.String("ReadOnly")}},
					},
				},
			},
		}, nil
	}

	loader := func(_ []string, _ time.Time) (SSOToken, error) {
		return SSOToken{
			AccessToken: "token",
			ExpiresAt:   time.Now().Add(time.Hour),
			IssuedAt:    time.Now().Add(-time.Minute),
			Region:      cfg.SSO.Region,
			StartURL:    cfg.SSO.StartURL,
		}, nil
	}

	profiles, err := discoverProfiles(context.Background(), cfg, factory, loader, time.Now())
	if err != nil {
		t.Fatalf("discoverProfiles failed: %v", err)
	}

	expected := []DiscoveredProfile{
		{
			AccountID:   "111111111111",
			AccountName: "prod",
			RoleName:    "Admin",
			SSORegion:   cfg.SSO.Region,
			SSOStartURL: cfg.SSO.StartURL,
		},
		{
			AccountID:   "111111111111",
			AccountName: "prod",
			RoleName:    "ReadOnly",
			SSORegion:   cfg.SSO.Region,
			SSOStartURL: cfg.SSO.StartURL,
		},
		{
			AccountID:   "222222222222",
			AccountName: "staging",
			RoleName:    "ReadOnly",
			SSORegion:   cfg.SSO.Region,
			SSOStartURL: cfg.SSO.StartURL,
		},
	}

	if !reflect.DeepEqual(profiles, expected) {
		t.Fatalf("profiles = %#v", profiles)
	}
}

func TestDiscoverProfilesErrors(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SSO.Region = "us-east-1"
	cfg.SSO.StartURL = "https://example.awsapps.com/start"
	cfg.TokenCachePaths = []string{"/cache"}

	loader := func(_ []string, _ time.Time) (SSOToken, error) {
		return SSOToken{}, errNoValidToken
	}

	_, err := discoverProfiles(context.Background(), cfg, nil, loader, time.Now())
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}
