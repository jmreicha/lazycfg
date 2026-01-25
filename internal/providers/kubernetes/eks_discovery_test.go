package kubernetes

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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
	credentialsData, err := readFixture("credentials_valid")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	credentialsPath := filepath.Join(tmpDir, "credentials")
	if err := writeFixture(credentialsPath, string(credentialsData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AWS.CredentialsFile = credentialsPath
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

	clusters, err := DiscoverEKSClusters(context.Background(), cfg, factory)
	if err != nil {
		t.Fatalf("DiscoverEKSClusters failed: %v", err)
	}

	if len(clusters) != 6 {
		t.Fatalf("clusters = %d", len(clusters))
	}

	expected := []DiscoveredCluster{
		{
			Profile:  "default",
			Region:   "us-east-1",
			Name:     "default-us-east-1",
			Endpoint: "https://default-us-east-1.example.com",
			CAData:   []byte("demo"),
		},
		{
			Profile:  "default",
			Region:   "us-west-2",
			Name:     "default-us-west-2",
			Endpoint: "https://default-us-west-2.example.com",
			CAData:   []byte("demo"),
		},
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
	credentialsPath := filepath.Join(tmpDir, "credentials")
	credentialsData, err := readFixture("credentials_empty")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	if err := writeFixture(credentialsPath, string(credentialsData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AWS.CredentialsFile = credentialsPath
	cfg.AWS.Regions = []string{"us-west-2"}

	_, err = DiscoverEKSClusters(context.Background(), cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty credentials file")
	}

	cfg.AWS.CredentialsFile = filepath.Join(tmpDir, "missing")
	_, err = DiscoverEKSClusters(context.Background(), cfg, nil)
	if err == nil {
		t.Fatal("expected error for missing credentials file")
	}
}

func TestResolveProfilesExplicit(t *testing.T) {
	profiles, err := resolveProfiles("", []string{"prod", "", "staging", "prod"})
	if err != nil {
		t.Fatalf("resolveProfiles failed: %v", err)
	}

	expected := []string{"prod", "staging"}
	if !reflect.DeepEqual(profiles, expected) {
		t.Fatalf("profiles = %v", profiles)
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
	_, err := DiscoverEKSClusters(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config, got nil")
	}
}

func TestDiscoverEKSClustersListError(t *testing.T) {
	tmpDir := t.TempDir()
	credentialsData, err := readFixture("credentials_valid")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	credentialsPath := filepath.Join(tmpDir, "credentials")
	if err := writeFixture(credentialsPath, string(credentialsData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AWS.CredentialsFile = credentialsPath
	cfg.AWS.Profiles = []string{"test"}
	cfg.AWS.Regions = []string{"us-west-2"}
	cfg.AWS.Timeout = 100 * time.Millisecond

	factory := func(_ context.Context, _, _ string) (EKSClient, error) {
		return &mockEKSClient{
			listErr: errors.New("list failed"),
		}, nil
	}

	_, err = DiscoverEKSClusters(context.Background(), cfg, factory)
	if err == nil {
		t.Fatal("expected error for list failure, got nil")
	}
}

func TestDiscoverEKSClustersFactoryError(t *testing.T) {
	tmpDir := t.TempDir()
	credentialsData, err := readFixture("credentials_valid")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	credentialsPath := filepath.Join(tmpDir, "credentials")
	if err := writeFixture(credentialsPath, string(credentialsData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	cfg := DefaultConfig()
	cfg.AWS.CredentialsFile = credentialsPath
	cfg.AWS.Profiles = []string{"test"}
	cfg.AWS.Regions = []string{"us-west-2"}
	cfg.AWS.Timeout = 100 * time.Millisecond

	factory := func(_ context.Context, _, _ string) (EKSClient, error) {
		return nil, errors.New("factory error")
	}

	_, err = DiscoverEKSClusters(context.Background(), cfg, factory)
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

func TestNormalizeProfiles(t *testing.T) {
	tests := []struct {
		name     string
		profiles []string
		expected []string
	}{
		{
			name:     "empty profiles",
			profiles: []string{},
			expected: []string{},
		},
		{
			name:     "nil profiles",
			profiles: nil,
			expected: []string{},
		},
		{
			name:     "profiles with whitespace",
			profiles: []string{" prod ", "  ", "staging"},
			expected: []string{"prod", "staging"},
		},
		{
			name:     "duplicate profiles",
			profiles: []string{"prod", "prod", "staging"},
			expected: []string{"prod", "staging"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeProfiles(tt.profiles)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("normalizeProfiles(%v) = %v, want %v", tt.profiles, result, tt.expected)
			}
		})
	}
}
