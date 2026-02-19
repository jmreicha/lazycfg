package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

type mockEKSClient struct {
	describeErr error
	listErr     error
	listOutput  *eks.ListClustersOutput
	outputs     map[string]*eks.DescribeClusterOutput
}

func (m *mockEKSClient) ListClusters(_ context.Context, _ *eks.ListClustersInput, _ ...func(*eks.Options)) (*eks.ListClustersOutput, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listOutput != nil {
		return m.listOutput, nil
	}
	return &eks.ListClustersOutput{}, nil
}

func (m *mockEKSClient) DescribeCluster(_ context.Context, params *eks.DescribeClusterInput, _ ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	if m.describeErr != nil {
		return nil, m.describeErr
	}
	if params == nil || params.Name == nil {
		return nil, errors.New("missing cluster name")
	}
	if output, ok := m.outputs[*params.Name]; ok {
		return output, nil
	}
	return nil, errors.New("cluster not found")
}

func TestDiscoverEKSClusters(t *testing.T) {
	tmpDir := t.TempDir()
	configData, err := readFixture("config_valid")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	configPath := filepath.Join(tmpDir, "config")
	if err := writeFixture(configPath, string(configData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AWS.ConfigFile = configPath
	cfg.AWS.CredentialsFile = ""
	cfg.AWS.Regions = []string{"us-west-2", "us-east-1"}
	cfg.AWS.ParallelWorkers = 2
	cfg.AWS.Timeout = 100 * time.Millisecond

	factory := func(_ context.Context, profile, region string) (EKSClient, error) {
		clusterName := profile + "-" + region
		endpoint := "https://" + clusterName + ".example.com"
		return &mockEKSClient{
			listOutput: &eks.ListClustersOutput{Clusters: []string{clusterName}},
			outputs: map[string]*eks.DescribeClusterOutput{
				clusterName: {
					Cluster: &types.Cluster{
						Name:     &clusterName,
						Endpoint: &endpoint,
						CertificateAuthority: &types.Certificate{
							Data: func() *string {
								value := "ZGVtbw=="
								return &value
							}(),
						},
					},
				},
			},
		}, nil
	}

	clusters, _, err := DiscoverEKSClusters(context.Background(), cfg, factory, nil)
	if err != nil {
		t.Fatalf("DiscoverEKSClusters failed: %v", err)
	}

	if len(clusters) != 4 {
		t.Fatalf("clusters = %d, want 4", len(clusters))
	}

	expected := []DiscoveredCluster{
		{
			Profile:  "prod",
			Region:   "us-east-1",
			Name:     "prod-us-east-1",
			Endpoint: "https://prod-us-east-1.example.com",
			CAData:   []byte("demo"),
		},
		{
			Profile:  "prod",
			Region:   "us-west-2",
			Name:     "prod-us-west-2",
			Endpoint: "https://prod-us-west-2.example.com",
			CAData:   []byte("demo"),
		},
		{
			Profile:  "staging",
			Region:   "us-east-1",
			Name:     "staging-us-east-1",
			Endpoint: "https://staging-us-east-1.example.com",
			CAData:   []byte("demo"),
		},
		{
			Profile:  "staging",
			Region:   "us-west-2",
			Name:     "staging-us-west-2",
			Endpoint: "https://staging-us-west-2.example.com",
			CAData:   []byte("demo"),
		},
	}

	if !reflect.DeepEqual(clusters, expected) {
		t.Fatalf("clusters = %#v", clusters)
	}
}

func TestDiscoverEKSClustersErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty config file with no profiles should fall back to credentials file.
	emptyConfigPath := filepath.Join(tmpDir, "config_empty")
	if err := writeFixture(emptyConfigPath, ""); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	credentialsData, err := readFixture("credentials_empty")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	credentialsPath := filepath.Join(tmpDir, "credentials")
	if err := writeFixture(credentialsPath, string(credentialsData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AWS.ConfigFile = emptyConfigPath
	cfg.AWS.CredentialsFile = credentialsPath
	cfg.AWS.Regions = []string{"us-west-2"}

	_, _, err = DiscoverEKSClusters(context.Background(), cfg, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty config and credentials files")
	}

	cfg.AWS.ConfigFile = filepath.Join(tmpDir, "missing")
	cfg.AWS.CredentialsFile = filepath.Join(tmpDir, "missing")
	_, _, err = DiscoverEKSClusters(context.Background(), cfg, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}

func TestNormalizeRegions(t *testing.T) {
	tests := []struct {
		name     string
		regions  []string
		expected []string
		wantErr  bool
	}{
		{
			name:    "nil regions",
			regions: nil,
			wantErr: true,
		},
		{
			name:    "empty regions",
			regions: []string{},
			wantErr: true,
		},
		{
			name:    "only whitespace regions",
			regions: []string{"  ", "", "   "},
			wantErr: true,
		},
		{
			name:     "valid regions",
			regions:  []string{"us-east-1", "us-west-2"},
			expected: []string{"us-east-1", "us-west-2"},
			wantErr:  false,
		},
		{
			name:     "regions with whitespace",
			regions:  []string{" us-east-1 ", "us-west-2"},
			expected: []string{"us-east-1", "us-west-2"},
			wantErr:  false,
		},
		{
			name:     "duplicate regions",
			regions:  []string{"us-east-1", "us-east-1", "us-west-2"},
			expected: []string{"us-east-1", "us-west-2"},
			wantErr:  false,
		},
		{
			name:     "regions with empty entries",
			regions:  []string{"us-east-1", "", "us-west-2"},
			expected: []string{"us-east-1", "us-west-2"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeRegions(tt.regions)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("normalizeRegions(%v) = %v, want %v", tt.regions, result, tt.expected)
			}
		})
	}
}

type mockRegionLister struct {
	output *ec2.DescribeRegionsOutput
	err    error
}

func (m *mockRegionLister) DescribeRegions(_ context.Context, _ *ec2.DescribeRegionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	return m.output, m.err
}

func TestResolveRegionsExplicit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	regions, err := resolveRegions(context.Background(), []string{"us-west-2", "eu-west-1"}, "", "test", nil, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"eu-west-1", "us-west-2"}
	if !reflect.DeepEqual(regions, expected) {
		t.Errorf("regions = %v, want %v", regions, expected)
	}
}

func TestResolveRegionsAllKeyword(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mock := &mockRegionLister{
		output: &ec2.DescribeRegionsOutput{
			Regions: []ec2types.Region{
				{RegionName: strPtr("us-east-1")},
				{RegionName: strPtr("eu-west-1")},
				{RegionName: strPtr("ap-southeast-1")},
			},
		},
	}

	oldFactory := regionListerFactory
	regionListerFactory = func(_ context.Context, _, _ string, _ map[string]*credentialProcessOutput) (RegionLister, error) {
		return mock, nil
	}
	defer func() { regionListerFactory = oldFactory }()

	regions, err := resolveRegions(context.Background(), []string{"all"}, "", "test-profile", nil, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"ap-southeast-1", "eu-west-1", "us-east-1"}
	if !reflect.DeepEqual(regions, expected) {
		t.Errorf("regions = %v, want %v", regions, expected)
	}
}

func TestResolveRegionsAllCaseInsensitive(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mock := &mockRegionLister{
		output: &ec2.DescribeRegionsOutput{
			Regions: []ec2types.Region{
				{RegionName: strPtr("us-east-1")},
			},
		},
	}

	oldFactory := regionListerFactory
	regionListerFactory = func(_ context.Context, _, _ string, _ map[string]*credentialProcessOutput) (RegionLister, error) {
		return mock, nil
	}
	defer func() { regionListerFactory = oldFactory }()

	for _, keyword := range []string{"ALL", "All", " all "} {
		regions, err := resolveRegions(context.Background(), []string{keyword}, "", "p", nil, logger)
		if err != nil {
			t.Fatalf("keyword %q: unexpected error: %v", keyword, err)
		}
		if !reflect.DeepEqual(regions, []string{"us-east-1"}) {
			t.Errorf("keyword %q: regions = %v", keyword, regions)
		}
	}
}

func TestResolveRegionsAllWithVaultCreds(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mock := &mockRegionLister{
		output: &ec2.DescribeRegionsOutput{
			Regions: []ec2types.Region{
				{RegionName: strPtr("us-west-2")},
			},
		},
	}

	var receivedCreds map[string]*credentialProcessOutput
	oldFactory := regionListerFactory
	regionListerFactory = func(_ context.Context, _, _ string, creds map[string]*credentialProcessOutput) (RegionLister, error) {
		receivedCreds = creds
		return mock, nil
	}
	defer func() { regionListerFactory = oldFactory }()

	vaultCreds := map[string]*credentialProcessOutput{
		"myprofile": {AccessKeyID: "AKIA", SecretAccessKey: "secret", SessionToken: "tok"},
	}

	regions, err := resolveRegions(context.Background(), []string{"all"}, "", "myprofile", vaultCreds, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(regions, []string{"us-west-2"}) {
		t.Errorf("regions = %v", regions)
	}
	if receivedCreds == nil {
		t.Fatal("expected vault creds to be passed through")
	}
	if _, ok := receivedCreds["myprofile"]; !ok {
		t.Error("expected myprofile in vault creds")
	}
}

func TestFetchAllRegionsError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	oldFactory := regionListerFactory
	regionListerFactory = func(_ context.Context, _, _ string, _ map[string]*credentialProcessOutput) (RegionLister, error) {
		return &mockRegionLister{err: errors.New("access denied")}, nil
	}
	defer func() { regionListerFactory = oldFactory }()

	_, err := resolveRegions(context.Background(), []string{"all"}, "", "p", nil, logger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "describe regions") {
		t.Errorf("error = %v, want describe regions error", err)
	}
}

func strPtr(s string) *string { return &s }

func TestDecodeClusterCA(t *testing.T) {
	tests := []struct {
		name     string
		ca       *types.Certificate
		expected []byte
		wantErr  bool
	}{
		{
			name:     "nil certificate",
			ca:       nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "nil data",
			ca:       &types.Certificate{Data: nil},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "empty data",
			ca: &types.Certificate{Data: func() *string {
				s := ""
				return &s
			}()},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "valid base64",
			ca: &types.Certificate{Data: func() *string {
				s := "dGVzdA=="
				return &s
			}()},
			expected: []byte("test"),
			wantErr:  false,
		},
		{
			name: "invalid base64",
			ca: &types.Certificate{Data: func() *string {
				s := "not-valid-base64!!!"
				return &s
			}()},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeClusterCA(tt.ca)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("decodeClusterCA() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDescribeClusterErrors(t *testing.T) {
	ctx := context.Background()
	timeout := 100 * time.Millisecond

	t.Run("describe error", func(t *testing.T) {
		client := &mockEKSClient{
			describeErr: errors.New("describe failed"),
		}

		_, err := describeCluster(ctx, client, timeout, "profile", "region", "cluster")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("nil cluster in response", func(t *testing.T) {
		client := &mockEKSClient{
			outputs: map[string]*eks.DescribeClusterOutput{
				"cluster": {Cluster: nil},
			},
		}

		_, err := describeCluster(ctx, client, timeout, "profile", "region", "cluster")
		if err == nil {
			t.Fatal("expected error for nil cluster, got nil")
		}
	})

	t.Run("missing endpoint", func(t *testing.T) {
		name := "cluster"
		client := &mockEKSClient{
			outputs: map[string]*eks.DescribeClusterOutput{
				name: {
					Cluster: &types.Cluster{
						Name:     &name,
						Endpoint: nil,
					},
				},
			},
		}

		_, err := describeCluster(ctx, client, timeout, "profile", "region", name)
		if err == nil {
			t.Fatal("expected error for missing endpoint, got nil")
		}
	})
}

func TestDiscoverEKSClustersNilConfig(t *testing.T) {
	_, _, err := DiscoverEKSClusters(context.Background(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config, got nil")
	}
}

func writeTestAWSConfig(t *testing.T, profiles ...string) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	var b strings.Builder
	for _, p := range profiles {
		fmt.Fprintf(&b, "[profile %s]\nregion = us-east-1\n\n", p)
	}
	if err := os.WriteFile(configPath, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("write test aws config: %v", err)
	}
	return configPath
}

func TestDiscoverEKSClustersListError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AWS.ConfigFile = writeTestAWSConfig(t, "test")
	cfg.AWS.Regions = []string{"us-west-2"}
	cfg.AWS.Timeout = 100 * time.Millisecond

	factory := func(_ context.Context, _, _ string) (EKSClient, error) {
		return &mockEKSClient{
			listErr: errors.New("list failed"),
		}, nil
	}

	_, _, err := DiscoverEKSClusters(context.Background(), cfg, factory, nil)
	if err == nil {
		t.Fatal("expected error for list failure, got nil")
	}
}

func TestDiscoverEKSClustersFactoryError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AWS.ConfigFile = writeTestAWSConfig(t, "test")
	cfg.AWS.Regions = []string{"us-west-2"}
	cfg.AWS.Timeout = 100 * time.Millisecond

	factory := func(_ context.Context, _, _ string) (EKSClient, error) {
		return nil, errors.New("factory error")
	}

	_, _, err := DiscoverEKSClusters(context.Background(), cfg, factory, nil)
	if err == nil {
		t.Fatal("expected error for factory failure, got nil")
	}
}

func TestSortClusters(t *testing.T) {
	clusters := []DiscoveredCluster{
		{Profile: "b", Region: "us-west-2", Name: "cluster1"},
		{Profile: "a", Region: "us-east-1", Name: "cluster2"},
		{Profile: "a", Region: "us-east-1", Name: "cluster1"},
		{Profile: "a", Region: "us-west-2", Name: "cluster1"},
	}

	sortClusters(clusters)

	expected := []DiscoveredCluster{
		{Profile: "a", Region: "us-east-1", Name: "cluster1"},
		{Profile: "a", Region: "us-east-1", Name: "cluster2"},
		{Profile: "a", Region: "us-west-2", Name: "cluster1"},
		{Profile: "b", Region: "us-west-2", Name: "cluster1"},
	}

	if !reflect.DeepEqual(clusters, expected) {
		t.Errorf("sortClusters did not produce expected order: got %+v", clusters)
	}
}

func TestParseCredentialProfiles(t *testing.T) {
	t.Run("empty credentials file path", func(t *testing.T) {
		_, err := parseCredentialProfiles("")
		if err == nil {
			t.Fatal("expected error for empty path, got nil")
		}
	})

	t.Run("whitespace credentials file path", func(t *testing.T) {
		_, err := parseCredentialProfiles("   ")
		if err == nil {
			t.Fatal("expected error for whitespace path, got nil")
		}
	})

	t.Run("credentials with comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		credentialsPath := filepath.Join(tmpDir, "credentials")
		content := `# Comment line
[default]
aws_access_key_id = default

; Another comment
[prod]
aws_access_key_id = prod
`
		if err := writeFixture(credentialsPath, content); err != nil {
			t.Fatalf("writeFixture failed: %v", err)
		}

		profiles, err := parseCredentialProfiles(credentialsPath)
		if err != nil {
			t.Fatalf("parseCredentialProfiles failed: %v", err)
		}

		expected := []string{"default", "prod"}
		if !reflect.DeepEqual(profiles, expected) {
			t.Errorf("profiles = %v, want %v", profiles, expected)
		}
	})
}

func TestParseConfigFileProfiles(t *testing.T) {
	t.Run("empty config file path", func(t *testing.T) {
		_, err := parseConfigFileProfiles("")
		if err == nil {
			t.Fatal("expected error for empty path, got nil")
		}
	})

	t.Run("whitespace config file path", func(t *testing.T) {
		_, err := parseConfigFileProfiles("   ")
		if err == nil {
			t.Fatal("expected error for whitespace path, got nil")
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config")
		content := `[sso-session cfgctl]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[profile prod]
sso_session = cfgctl
sso_account_id = 111111111111
sso_role_name = AdminAccess

[profile staging]
sso_session = cfgctl
sso_account_id = 222222222222
sso_role_name = AdminAccess

[default]
region = us-east-1
`
		if err := writeFixture(configPath, content); err != nil {
			t.Fatalf("writeFixture failed: %v", err)
		}

		profiles, err := parseConfigFileProfiles(configPath)
		if err != nil {
			t.Fatalf("parseConfigFileProfiles failed: %v", err)
		}

		expected := []string{"prod", "staging"}
		if !reflect.DeepEqual(profiles, expected) {
			t.Errorf("profiles = %v, want %v", profiles, expected)
		}
	})

	t.Run("config file with comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config")
		content := `# Comment line
[sso-session test]
sso_start_url = https://example.awsapps.com/start

; Another comment
[profile dev]
sso_session = test

[profile prod]
sso_session = test
`
		if err := writeFixture(configPath, content); err != nil {
			t.Fatalf("writeFixture failed: %v", err)
		}

		profiles, err := parseConfigFileProfiles(configPath)
		if err != nil {
			t.Fatalf("parseConfigFileProfiles failed: %v", err)
		}

		expected := []string{"dev", "prod"}
		if !reflect.DeepEqual(profiles, expected) {
			t.Errorf("profiles = %v, want %v", profiles, expected)
		}
	})

	t.Run("missing config file", func(t *testing.T) {
		_, err := parseConfigFileProfiles("/nonexistent/path")
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("empty config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config")
		if err := writeFixture(configPath, ""); err != nil {
			t.Fatalf("writeFixture failed: %v", err)
		}

		profiles, err := parseConfigFileProfiles(configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(profiles) != 0 {
			t.Errorf("profiles = %v, want empty", profiles)
		}
	})
}

func TestFilterProfilesByRole(t *testing.T) {
	profiles := []string{
		"prod/adminaccess",
		"prod/readonly",
		"staging/adminaccess",
		"staging/poweruser",
		"simple-profile",
	}

	tests := []struct {
		name     string
		roles    []string
		expected []string
	}{
		{
			name:     "single role",
			roles:    []string{"adminaccess"},
			expected: []string{"prod/adminaccess", "staging/adminaccess"},
		},
		{
			name:     "multiple roles",
			roles:    []string{"adminaccess", "readonly"},
			expected: []string{"prod/adminaccess", "prod/readonly", "staging/adminaccess"},
		},
		{
			name:     "case insensitive",
			roles:    []string{"AdminAccess"},
			expected: []string{"prod/adminaccess", "staging/adminaccess"},
		},
		{
			name:     "no slash in profile",
			roles:    []string{"simple-profile"},
			expected: []string{"simple-profile"},
		},
		{
			name:     "no match",
			roles:    []string{"nonexistent"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterProfilesByRole(profiles, tt.roles)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterProfilesByRole() = %v, want %v", result, tt.expected)
			}
		})
	}
}
