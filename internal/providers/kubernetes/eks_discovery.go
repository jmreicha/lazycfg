package kubernetes

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"golang.org/x/sync/errgroup"
)

var errProfilesNotFound = errors.New("no aws profiles found in credentials file")

// EKSClient defines the AWS EKS client interface for cluster discovery.
type EKSClient interface {
	ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error)
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
}

// EKSClientFactory builds an EKS client for the provided profile and region.
type EKSClientFactory func(ctx context.Context, profile, region string) (EKSClient, error)

// DiscoverEKSClusters scans AWS profiles and regions for EKS clusters.
func DiscoverEKSClusters(ctx context.Context, cfg *Config, factory EKSClientFactory) ([]DiscoveredCluster, error) {
	if cfg == nil {
		return nil, errors.New("kubernetes config is nil")
	}

	if factory == nil {
		factory = NewEKSClientFactory(cfg.AWS.CredentialsFile)
	}

	profiles, err := resolveProfiles(cfg.AWS.CredentialsFile, cfg.AWS.Profiles)
	if err != nil {
		return nil, err
	}

	regions, err := normalizeRegions(cfg.AWS.Regions)
	if err != nil {
		return nil, err
	}

	g, groupCtx := errgroup.WithContext(ctx)
	if cfg.AWS.ParallelWorkers > 0 {
		g.SetLimit(cfg.AWS.ParallelWorkers)
	}

	var (
		clusters []DiscoveredCluster
		mu       sync.Mutex
	)

	for _, profile := range profiles {
		for _, region := range regions {
			profile := profile
			g.Go(func() error {
				return discoverClustersForProfileRegion(groupCtx, cfg, factory, profile, region, &mu, &clusters)
			})
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sortClusters(clusters)

	return clusters, nil
}

func discoverClustersForProfileRegion(ctx context.Context, cfg *Config, factory EKSClientFactory, profile, region string, mu *sync.Mutex, clusters *[]DiscoveredCluster) error {
	client, err := factory(ctx, profile, region)
	if err != nil {
		return fmt.Errorf("create eks client for profile %q region %q: %w", profile, region, err)
	}

	listCtx, cancel := context.WithTimeout(ctx, cfg.AWS.Timeout)
	defer cancel()

	listOutput, err := client.ListClusters(listCtx, &eks.ListClustersInput{})
	if err != nil {
		return fmt.Errorf("list eks clusters for profile %q region %q: %w", profile, region, err)
	}

	for _, name := range listOutput.Clusters {
		cluster, err := describeCluster(ctx, client, cfg.AWS.Timeout, profile, region, name)
		if err != nil {
			return err
		}

		mu.Lock()
		*clusters = append(*clusters, cluster)
		mu.Unlock()
	}

	return nil
}

// NewEKSClientFactory returns a default EKS client factory.
func NewEKSClientFactory(credentialsFile string) EKSClientFactory {
	return func(ctx context.Context, profile, region string) (EKSClient, error) {
		loadOpts := []func(*config.LoadOptions) error{
			config.WithRegion(region),
			config.WithSharedConfigProfile(profile),
		}

		if credentialsFile != "" {
			loadOpts = append(loadOpts, config.WithSharedCredentialsFiles([]string{credentialsFile}))
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

func resolveProfiles(credentialsFile string, profiles []string) ([]string, error) {
	if len(profiles) > 0 {
		return normalizeProfiles(profiles), nil
	}

	parsed, err := parseCredentialProfiles(credentialsFile)
	if err != nil {
		return nil, err
	}

	if len(parsed) == 0 {
		return nil, errProfilesNotFound
	}

	return parsed, nil
}

func normalizeProfiles(profiles []string) []string {
	set := make(map[string]struct{})
	for _, profile := range profiles {
		profile = strings.TrimSpace(profile)
		if profile == "" {
			continue
		}
		set[profile] = struct{}{}
	}

	return sortedKeys(set)
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
