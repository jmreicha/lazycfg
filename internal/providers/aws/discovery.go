package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

var errSSOLoginRequired = errors.New("sso session missing or expired, run 'aws sso login' to refresh")

// DiscoveredProfile represents an account/role from SSO.
type DiscoveredProfile struct {
	AccountID   string
	AccountName string
	RoleName    string
	SSORegion   string
	SSOStartURL string
}

// DiscoverProfiles uses the AWS SSO API to enumerate accounts and roles.
func DiscoverProfiles(ctx context.Context, cfg *Config, factory SSOClientFactory) ([]DiscoveredProfile, error) {
	loader := func(cachePaths []string, startURL, region string, now time.Time) (SSOToken, error) {
		return LoadMatchingToken(cachePaths, startURL, region, now)
	}
	return discoverProfiles(ctx, cfg, factory, loader, time.Now().UTC())
}

type tokenLoader func(cachePaths []string, startURL, region string, now time.Time) (SSOToken, error)

func discoverProfiles(ctx context.Context, cfg *Config, factory SSOClientFactory, loader tokenLoader, now time.Time) ([]DiscoveredProfile, error) {
	if cfg == nil {
		return nil, errors.New("aws config is nil")
	}

	if cfg.Demo {
		return demoProfiles(cfg), nil
	}

	if factory == nil {
		factory = NewSSOClientFactory()
	}

	token, err := loader(cfg.TokenCachePaths, cfg.SSO.StartURL, cfg.SSO.Region, now)
	if err != nil {
		if errors.Is(err, errNoValidToken) {
			return nil, errSSOLoginRequired
		}
		return nil, err
	}

	client, err := factory(ctx, cfg.SSO.Region, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("create sso client: %w", err)
	}

	roleFilter := normalizeRoleFilter(cfg.Roles)
	accounts, err := listAccounts(ctx, client, token.AccessToken)
	if err != nil {
		return nil, err
	}

	profiles := []DiscoveredProfile{}
	for _, account := range accounts {
		roles, err := listAccountRoles(ctx, client, token.AccessToken, account)
		if err != nil {
			return nil, err
		}
		for _, role := range roles {
			if !roleFilter[role] && len(roleFilter) > 0 {
				continue
			}
			profiles = append(profiles, DiscoveredProfile{
				AccountID:   aws.ToString(account.AccountId),
				AccountName: aws.ToString(account.AccountName),
				RoleName:    role,
				SSORegion:   cfg.SSO.Region,
				SSOStartURL: cfg.SSO.StartURL,
			})
		}
	}

	sortProfiles(profiles)
	return profiles, nil
}

func demoProfiles(cfg *Config) []DiscoveredProfile {
	return []DiscoveredProfile{
		{
			AccountID:   "111111111111",
			AccountName: "demo",
			RoleName:    "AdminAccess",
			SSORegion:   cfg.SSO.Region,
			SSOStartURL: cfg.SSO.StartURL,
		},
		{
			AccountID:   "111111111111",
			AccountName: "demo",
			RoleName:    "ReadOnly",
			SSORegion:   cfg.SSO.Region,
			SSOStartURL: cfg.SSO.StartURL,
		},
		{
			AccountID:   "222222222222",
			AccountName: "sandbox",
			RoleName:    "AdminAccess",
			SSORegion:   cfg.SSO.Region,
			SSOStartURL: cfg.SSO.StartURL,
		},
	}
}

func listAccounts(ctx context.Context, client SSOClient, accessToken string) ([]types.AccountInfo, error) {
	input := &sso.ListAccountsInput{AccessToken: aws.String(accessToken)}
	accounts := []types.AccountInfo{}

	for {
		output, err := client.ListAccounts(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list sso accounts: %w", err)
		}
		accounts = append(accounts, output.AccountList...)
		if output.NextToken == nil || aws.ToString(output.NextToken) == "" {
			break
		}
		input.NextToken = output.NextToken
	}

	return accounts, nil
}

func listAccountRoles(ctx context.Context, client SSOClient, accessToken string, account types.AccountInfo) ([]string, error) {
	if account.AccountId == nil || aws.ToString(account.AccountId) == "" {
		return nil, errors.New("account id missing")
	}

	input := &sso.ListAccountRolesInput{
		AccessToken: aws.String(accessToken),
		AccountId:   account.AccountId,
	}
	roles := []string{}

	for {
		output, err := client.ListAccountRoles(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list account roles for %s: %w", aws.ToString(account.AccountId), err)
		}
		for _, role := range output.RoleList {
			name := strings.TrimSpace(aws.ToString(role.RoleName))
			if name == "" {
				continue
			}
			roles = append(roles, name)
		}
		if output.NextToken == nil || aws.ToString(output.NextToken) == "" {
			break
		}
		input.NextToken = output.NextToken
	}

	sort.Strings(roles)
	return roles, nil
}

func normalizeRoleFilter(roles []string) map[string]bool {
	filter := make(map[string]bool)
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		filter[trimmed] = true
	}

	return filter
}

func sortProfiles(profiles []DiscoveredProfile) {
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].AccountName != profiles[j].AccountName {
			return profiles[i].AccountName < profiles[j].AccountName
		}
		if profiles[i].AccountID != profiles[j].AccountID {
			return profiles[i].AccountID < profiles[j].AccountID
		}
		return profiles[i].RoleName < profiles[j].RoleName
	})
}
