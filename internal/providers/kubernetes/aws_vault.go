package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

// awsVaultCommand is the command name for aws-vault. Overridden in tests.
var awsVaultCommand = "aws-vault"

// credentialProcessOutput represents the credential_process JSON output
// from aws-vault exec --json.
type credentialProcessOutput struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
}

func (c *credentialProcessOutput) validate() error {
	if strings.TrimSpace(c.AccessKeyID) == "" {
		return errors.New("aws-vault returned empty AccessKeyId")
	}
	if strings.TrimSpace(c.SecretAccessKey) == "" {
		return errors.New("aws-vault returned empty SecretAccessKey")
	}
	return nil
}

// isAWSVaultAvailable reports whether aws-vault is on PATH.
func isAWSVaultAvailable() bool {
	_, err := exec.LookPath(awsVaultCommand)
	return err == nil
}

// fetchAWSVaultCredentials calls aws-vault exec <profile> --json
// and returns the parsed credentials.
func fetchAWSVaultCredentials(ctx context.Context, profile string) (*credentialProcessOutput, error) {
	// #nosec G204 -- profile is from user configuration, not external input.
	cmd := exec.CommandContext(ctx, awsVaultCommand, "exec", profile, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws-vault exec for profile %q: %w", profile, err)
	}

	var creds credentialProcessOutput
	if err := json.Unmarshal(output, &creds); err != nil {
		return nil, fmt.Errorf("parse aws-vault output for profile %q: %w", profile, err)
	}

	if err := creds.validate(); err != nil {
		return nil, fmt.Errorf("invalid credentials for profile %q: %w", profile, err)
	}

	return &creds, nil
}

// prefetchAWSVaultCredentials fetches credentials for all profiles serially.
// Serial execution ensures only the first call triggers SSO login;
// subsequent calls reuse the cached SSO token from the keychain.
func prefetchAWSVaultCredentials(ctx context.Context, profiles []string) (map[string]*credentialProcessOutput, error) {
	creds := make(map[string]*credentialProcessOutput, len(profiles))

	for _, profile := range profiles {
		cred, err := fetchAWSVaultCredentials(ctx, profile)
		if err != nil {
			return nil, err
		}
		creds[profile] = cred
	}

	return creds, nil
}

// NewAWSVaultEKSClientFactory returns a factory that creates EKS clients
// using pre-fetched aws-vault credentials. Credentials are looked up by
// profile and reused across all regions.
func NewAWSVaultEKSClientFactory(creds map[string]*credentialProcessOutput) EKSClientFactory {
	return func(ctx context.Context, profile, region string) (EKSClient, error) {
		cred, ok := creds[profile]
		if !ok {
			return nil, fmt.Errorf("no cached aws-vault credentials for profile %q", profile)
		}

		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					cred.AccessKeyID,
					cred.SecretAccessKey,
					cred.SessionToken,
				),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("load aws config for profile %q region %q: %w", profile, region, err)
		}

		return eks.NewFromConfig(cfg), nil
	}
}
