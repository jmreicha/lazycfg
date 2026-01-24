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
