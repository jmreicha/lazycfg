// Package aws provides AWS provider implementations.
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

// SSOClient defines the interface for AWS SSO operations.
type SSOClient interface {
	ListAccounts(ctx context.Context, params *sso.ListAccountsInput, optFns ...func(*sso.Options)) (*sso.ListAccountsOutput, error)
	ListAccountRoles(ctx context.Context, params *sso.ListAccountRolesInput, optFns ...func(*sso.Options)) (*sso.ListAccountRolesOutput, error)
}

// SSOClientFactory creates SSO clients with the appropriate credentials.
type SSOClientFactory func(ctx context.Context, region, accessToken string) (SSOClient, error)

// NewSSOClientFactory returns a default SSO client factory.
func NewSSOClientFactory() SSOClientFactory {
	return func(ctx context.Context, region, accessToken string) (SSOClient, error) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			return nil, fmt.Errorf("load aws config: %w", err)
		}

		client := sso.NewFromConfig(cfg, func(opts *sso.Options) {
			opts.Credentials = credentials.NewStaticCredentialsProvider("", "", accessToken)
		})

		return client, nil
	}
}
