package kubernetes

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/smithy-go"
	awsprovider "github.com/jmreicha/cfgctl/internal/providers/aws"
	"golang.org/x/sync/errgroup"
)

const authModeAWSVault = "aws-vault"

var errProfilesNotFound = errors.New("no aws profiles found in credentials file")

// RegionLister describes the EC2 DescribeRegions API.
type RegionLister interface {
	DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
}

// regionListerFactory builds a RegionLister. Override in tests.
var regionListerFactory func(ctx context.Context, configFile, profile string, vaultCreds map[string]*credentialProcessOutput) (RegionLister, error)

// EKSClient defines the AWS EKS client interface for cluster discovery.
type EKSClient interface {
	ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error)
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
}

// EKSClientFactory builds an EKS client for the provided profile and region.
type EKSClientFactory func(ctx context.Context, profile, region string) (EKSClient, error)

// DiscoverEKSClusters scans AWS profiles and regions for EKS clusters.
func DiscoverEKSClusters(ctx context.Context, cfg *Config, factory EKSClientFactory, logger *slog.Logger) ([]DiscoveredCluster, []string, error) {
	if cfg == nil {
		return nil, nil, errors.New("kubernetes config is nil")
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	profiles, err := resolveProfiles(cfg.AWS.ConfigFile, cfg.AWS.CredentialsFile)
	if err != nil {
		return nil, nil, err
	}
	logger.Debug("resolved aws profiles", "count", len(profiles), "profiles", profiles)

	if len(cfg.AWS.Roles) > 0 {
		profiles = filterProfilesByRole(profiles, cfg.AWS.Roles)
		logger.Debug("filtered profiles by role", "roles", cfg.AWS.Roles, "remaining", len(profiles))
	}

	var (
		authMode   string
		vaultCreds map[string]*credentialProcessOutput
	)
	if factory == nil {
		switch {
		case hasValidSSOToken():
			factory = NewEKSClientFactory(cfg.AWS.ConfigFile)
			logger.Debug("using SSO token auth")
		case isAWSVaultAvailable():
			logger.Debug("using aws-vault auth")
			var err error
			vaultCreds, err = prefetchAWSVaultCredentials(ctx, profiles)
			if err != nil {
				return nil, nil, fmt.Errorf("prefetch aws-vault credentials: %w", err)
			}
			factory = NewAWSVaultEKSClientFactory(vaultCreds)
			authMode = authModeAWSVault
		default:
			factory = NewEKSClientFactory(cfg.AWS.ConfigFile)
			logger.Debug("using default aws auth")
		}
	}

	regions, err := resolveRegions(ctx, cfg.AWS.Regions, cfg.AWS.ConfigFile, profiles[0], vaultCreds, logger)
	if err != nil {
		return nil, nil, err
	}
	logger.Debug("scanning regions", "count", len(regions), "regions", regions)

	g, groupCtx := errgroup.WithContext(ctx)
	if cfg.AWS.ParallelWorkers > 0 {
		g.SetLimit(cfg.AWS.ParallelWorkers)
	}

	var (
		clusters []DiscoveredCluster
		warnings []string
		mu       sync.Mutex
	)

	for _, profile := range profiles {
		for _, region := range regions {
			profile := profile
			g.Go(func() error {
				return discoverClustersForProfileRegion(groupCtx, cfg, factory, profile, region, authMode, &mu, &clusters, &warnings)
			})
		}
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	sortClusters(clusters)
	sort.Strings(warnings)

	return clusters, warnings, nil
}

func discoverClustersForProfileRegion(ctx context.Context, cfg *Config, factory EKSClientFactory, profile, region, authMode string, mu *sync.Mutex, clusters *[]DiscoveredCluster, warnings *[]string) error {
	client, err := factory(ctx, profile, region)
	if err != nil {
		return fmt.Errorf("create eks client for profile %q region %q: %w", profile, region, err)
	}

	listCtx, cancel := context.WithTimeout(ctx, cfg.AWS.Timeout)
	defer cancel()

	listOutput, err := client.ListClusters(listCtx, &eks.ListClustersInput{})
	if err != nil {
		if isAccessDenied(err) {
			mu.Lock()
			*warnings = append(*warnings, fmt.Sprintf("skipping profile %q region %q: access denied for eks:ListClusters", profile, region))
			mu.Unlock()
			return nil
		}
		return fmt.Errorf("list eks clusters for profile %q region %q: %w", profile, region, err)
	}

	for _, name := range listOutput.Clusters {
		cluster, err := describeCluster(ctx, client, cfg.AWS.Timeout, profile, region, name)
		if err != nil {
			if isAccessDenied(err) {
				mu.Lock()
				*warnings = append(*warnings, fmt.Sprintf("skipping cluster %q for profile %q region %q: access denied", name, profile, region))
				mu.Unlock()
				continue
			}
			return err
		}
		cluster.AuthMode = authMode

		mu.Lock()
		*clusters = append(*clusters, cluster)
		mu.Unlock()
	}

	return nil
}

// NewEKSClientFactory returns a default EKS client factory that loads
// credentials from the AWS config file using native SSO authentication.
// The credentials file is suppressed to avoid triggering credential_process
// helpers (e.g. aws-vault, granted) for each profile.
func NewEKSClientFactory(configFile string) EKSClientFactory {
	return func(ctx context.Context, profile, region string) (EKSClient, error) {
		loadOpts := []func(*config.LoadOptions) error{
			config.WithRegion(region),
			config.WithSharedConfigProfile(profile),
		}

		if configFile != "" {
			loadOpts = append(loadOpts,
				config.WithSharedConfigFiles([]string{configFile}),
				// Suppress credentials file to prevent credential_process invocation.
				config.WithSharedCredentialsFiles([]string{os.DevNull}),
			)
		}

		cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
		if err != nil {
			return nil, fmt.Errorf("load aws config for profile %q region %q: %w", profile, region, err)
		}

		return eks.NewFromConfig(cfg), nil
	}
}

func describeCluster(ctx context.Context, client EKSClient, timeout time.Duration, profile, region, name string) (DiscoveredCluster, error) {
	describeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := client.DescribeCluster(describeCtx, &eks.DescribeClusterInput{
		Name: aws.String(name),
	})
	if err != nil {
		return DiscoveredCluster{}, fmt.Errorf("describe eks cluster %q for profile %q region %q: %w", name, profile, region, err)
	}

	if output.Cluster == nil {
		return DiscoveredCluster{}, fmt.Errorf("describe eks cluster %q for profile %q region %q returned nil cluster", name, profile, region)
	}

	cluster := output.Cluster
	if cluster.Endpoint == nil {
		return DiscoveredCluster{}, fmt.Errorf("describe eks cluster %q for profile %q region %q missing endpoint", name, profile, region)
	}

	caData, err := decodeClusterCA(cluster.CertificateAuthority)
	if err != nil {
		return DiscoveredCluster{}, fmt.Errorf("decode eks cluster %q certificate authority: %w", name, err)
	}

	return DiscoveredCluster{
		Profile:  profile,
		Region:   region,
		Name:     name,
		Endpoint: aws.ToString(cluster.Endpoint),
		CAData:   caData,
	}, nil
}

// filterProfilesByRole returns only profiles whose role segment matches one of
// the given roles. The role is extracted as the last path component of the
// profile name (e.g. "prod/adminaccess" → "adminaccess"). Matching is
// case-insensitive.
func filterProfilesByRole(profiles, roles []string) []string {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}

	var filtered []string
	for _, profile := range profiles {
		role := profile
		if idx := strings.LastIndex(profile, "/"); idx >= 0 {
			role = profile[idx+1:]
		}
		if _, ok := roleSet[strings.ToLower(role)]; ok {
			filtered = append(filtered, profile)
		}
	}
	return filtered
}

// isAccessDenied reports whether an error is an AWS AccessDeniedException.
func isAccessDenied(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "AccessDeniedException"
	}
	return false
}

// hasValidSSOToken checks if there is a valid SSO token in the default cache.
func hasValidSSOToken() bool {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	cachePaths := []string{filepath.Join(home, ".aws", "sso", "cache")}
	_, err = awsprovider.LoadNewestToken(cachePaths, time.Now())
	return err == nil
}

func decodeClusterCA(authority *types.Certificate) ([]byte, error) {
	if authority == nil || authority.Data == nil || *authority.Data == "" {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(aws.ToString(authority.Data))
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func resolveProfiles(configFile, credentialsFile string) ([]string, error) {
	// Prefer config file for profile resolution (SSO-based profiles).
	if strings.TrimSpace(configFile) != "" {
		parsed, err := parseConfigFileProfiles(configFile)
		if err == nil && len(parsed) > 0 {
			return parsed, nil
		}
	}

	// Fall back to credentials file.
	parsed, err := parseCredentialProfiles(credentialsFile)
	if err != nil {
		return nil, err
	}

	if len(parsed) == 0 {
		return nil, errProfilesNotFound
	}

	return parsed, nil
}

func parseCredentialProfiles(credentialsFile string) (profiles []string, err error) {
	if strings.TrimSpace(credentialsFile) == "" {
		return nil, errAWSCredentialsEmpty
	}

	// #nosec G304 -- credentials file path is user configured.
	file, err := os.Open(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("open aws credentials file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close aws credentials file: %w", closeErr)
		}
	}()

	profileSet := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}

		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
		name = strings.TrimPrefix(name, "profile ")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		profileSet[name] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read aws credentials file: %w", err)
	}

	return sortedKeys(profileSet), nil
}

// parseConfigFileProfiles extracts profile names from an AWS config file.
// Only [profile xxx] sections are included; other sections like [sso-session ...]
// and [default] are skipped.
func parseConfigFileProfiles(configFile string) (profiles []string, err error) {
	if strings.TrimSpace(configFile) == "" {
		return nil, errAWSConfigFileEmpty
	}

	// #nosec G304 -- config file path is user configured.
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("open aws config file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close aws config file: %w", closeErr)
		}
	}()

	profileSet := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}

		section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))

		// Only include [profile xxx] sections.
		if !strings.HasPrefix(section, "profile ") {
			continue
		}

		name := strings.TrimSpace(strings.TrimPrefix(section, "profile "))
		if name == "" {
			continue
		}

		profileSet[name] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read aws config file: %w", err)
	}

	return sortedKeys(profileSet), nil
}

func resolveRegions(ctx context.Context, regions []string, configFile, profile string, vaultCreds map[string]*credentialProcessOutput, logger *slog.Logger) ([]string, error) {
	if len(regions) == 0 {
		return nil, errRegionsEmpty
	}

	for _, r := range regions {
		if strings.EqualFold(strings.TrimSpace(r), "all") {
			logger.Debug("fetching all enabled AWS regions", "profile", profile)
			return fetchAllRegions(ctx, configFile, profile, vaultCreds)
		}
	}

	return normalizeRegions(regions)
}

func normalizeRegions(regions []string) ([]string, error) {
	if len(regions) == 0 {
		return nil, errRegionsEmpty
	}

	set := make(map[string]struct{})
	for _, region := range regions {
		trimmed := strings.TrimSpace(region)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}

	if len(set) == 0 {
		return nil, errRegionsEmpty
	}

	return sortedKeys(set), nil
}

func fetchAllRegions(ctx context.Context, configFile, profile string, vaultCreds map[string]*credentialProcessOutput) ([]string, error) {
	var client RegionLister

	if regionListerFactory != nil {
		var err error
		client, err = regionListerFactory(ctx, configFile, profile, vaultCreds)
		if err != nil {
			return nil, err
		}
	} else {
		opts := []func(*config.LoadOptions) error{
			config.WithRegion("us-east-1"),
		}

		if cred, ok := vaultCreds[profile]; ok {
			opts = append(opts, config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					cred.AccessKeyID,
					cred.SecretAccessKey,
					cred.SessionToken,
				),
			))
		} else {
			opts = append(opts, config.WithSharedConfigProfile(profile))
			if configFile != "" {
				opts = append(opts,
					config.WithSharedConfigFiles([]string{configFile}),
					config.WithSharedCredentialsFiles([]string{os.DevNull}),
				)
			}
		}

		awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("load aws config for region discovery: %w", err)
		}

		client = ec2.NewFromConfig(awsCfg)
	}

	out, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false),
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("opt-in-status"),
				Values: []string{"opt-in-not-required", "opted-in"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("describe regions: %w", err)
	}

	regions := make([]string, 0, len(out.Regions))
	for _, r := range out.Regions {
		if r.RegionName != nil {
			regions = append(regions, *r.RegionName)
		}
	}
	sort.Strings(regions)
	return regions, nil
}

func sortClusters(clusters []DiscoveredCluster) {
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Profile != clusters[j].Profile {
			return clusters[i].Profile < clusters[j].Profile
		}
		if clusters[i].Region != clusters[j].Region {
			return clusters[i].Region < clusters[j].Region
		}
		return clusters[i].Name < clusters[j].Name
	})
}

func sortedKeys(items map[string]struct{}) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}
