package kubernetes

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func TestIsAWSVaultAvailable(t *testing.T) {
	t.Run("available when on PATH", func(t *testing.T) {
		// "ls" is always available on macOS/Linux.
		old := awsVaultCommand
		awsVaultCommand = "ls"
		defer func() { awsVaultCommand = old }()

		if !isAWSVaultAvailable() {
			t.Error("expected true when command is on PATH")
		}
	})

	t.Run("not available when missing", func(t *testing.T) {
		old := awsVaultCommand
		awsVaultCommand = "nonexistent-command-that-does-not-exist"
		defer func() { awsVaultCommand = old }()

		if isAWSVaultAvailable() {
			t.Error("expected false when command is not on PATH")
		}
	})
}

func TestCredentialProcessOutputValidate(t *testing.T) {
	tests := []struct {
		name    string
		cred    credentialProcessOutput
		wantErr bool
	}{
		{
			name: "valid credentials",
			cred: credentialProcessOutput{
				Version:         1,
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "token",
			},
			wantErr: false,
		},
		{
			name: "empty access key",
			cred: credentialProcessOutput{
				Version:         1,
				AccessKeyID:     "",
				SecretAccessKey: "secret",
			},
			wantErr: true,
		},
		{
			name: "empty secret key",
			cred: credentialProcessOutput{
				Version:         1,
				AccessKeyID:     "AKID",
				SecretAccessKey: "",
			},
			wantErr: true,
		},
		{
			name: "whitespace access key",
			cred: credentialProcessOutput{
				Version:         1,
				AccessKeyID:     "   ",
				SecretAccessKey: "secret",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cred.validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCredentialProcessOutputJSON(t *testing.T) {
	jsonData := `{
		"Version": 1,
		"AccessKeyId": "AKIAIOSFODNN7EXAMPLE",
		"SecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"SessionToken": "FwoGZXIvYXdzEBY",
		"Expiration": "2025-01-01T00:00:00Z"
	}`

	var cred credentialProcessOutput
	if err := json.Unmarshal([]byte(jsonData), &cred); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if cred.Version != 1 {
		t.Errorf("Version = %d, want 1", cred.Version)
	}
	if cred.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("AccessKeyID = %q", cred.AccessKeyID)
	}
	if cred.SecretAccessKey != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("SecretAccessKey = %q", cred.SecretAccessKey)
	}
	if cred.SessionToken != "FwoGZXIvYXdzEBY" {
		t.Errorf("SessionToken = %q", cred.SessionToken)
	}
	if cred.Expiration != "2025-01-01T00:00:00Z" {
		t.Errorf("Expiration = %q", cred.Expiration)
	}

	if err := cred.validate(); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}

func TestFetchAWSVaultCredentials(t *testing.T) {
	// Create a mock script that outputs credential_process JSON.
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-aws-vault")

	creds := credentialProcessOutput{
		Version:         1,
		AccessKeyID:     "AKIATEST",
		SecretAccessKey: "SECRET",
		SessionToken:    "TOKEN",
		Expiration:      "2099-01-01T00:00:00Z",
	}
	jsonBytes, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Write a shell script that outputs the JSON regardless of arguments.
	scriptContent := "#!/bin/sh\necho '" + string(jsonBytes) + "'\n"
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	old := awsVaultCommand
	awsVaultCommand = mockScript
	defer func() { awsVaultCommand = old }()

	result, err := fetchAWSVaultCredentials(context.Background(), "test-profile")
	if err != nil {
		t.Fatalf("fetchAWSVaultCredentials failed: %v", err)
	}

	if result.AccessKeyID != "AKIATEST" {
		t.Errorf("AccessKeyID = %q", result.AccessKeyID)
	}
	if result.SecretAccessKey != "SECRET" {
		t.Errorf("SecretAccessKey = %q", result.SecretAccessKey)
	}
	if result.SessionToken != "TOKEN" {
		t.Errorf("SessionToken = %q", result.SessionToken)
	}
}

func TestFetchAWSVaultCredentialsError(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-aws-vault")

	scriptContent := "#!/bin/sh\necho 'error' >&2\nexit 1\n"
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	old := awsVaultCommand
	awsVaultCommand = mockScript
	defer func() { awsVaultCommand = old }()

	_, err := fetchAWSVaultCredentials(context.Background(), "test-profile")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPrefetchAWSVaultCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-aws-vault")

	creds := credentialProcessOutput{
		Version:         1,
		AccessKeyID:     "AKIATEST",
		SecretAccessKey: "SECRET",
		SessionToken:    "TOKEN",
		Expiration:      "2099-01-01T00:00:00Z",
	}
	jsonBytes, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	scriptContent := "#!/bin/sh\necho '" + string(jsonBytes) + "'\n"
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	old := awsVaultCommand
	awsVaultCommand = mockScript
	defer func() { awsVaultCommand = old }()

	result, err := prefetchAWSVaultCredentials(context.Background(), []string{"prod", "staging"})
	if err != nil {
		t.Fatalf("prefetchAWSVaultCredentials failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("result count = %d, want 2", len(result))
	}

	for _, profile := range []string{"prod", "staging"} {
		cred, ok := result[profile]
		if !ok {
			t.Fatalf("missing credentials for profile %q", profile)
		}
		if cred.AccessKeyID != "AKIATEST" {
			t.Errorf("profile %q AccessKeyID = %q", profile, cred.AccessKeyID)
		}
	}
}

func TestNewAWSVaultEKSClientFactory(t *testing.T) {
	creds := map[string]*credentialProcessOutput{
		"prod": {
			AccessKeyID:     "AKIAPROD",
			SecretAccessKey: "SECRETPROD",
			SessionToken:    "TOKENPROD",
		},
	}

	factory := NewAWSVaultEKSClientFactory(creds)

	t.Run("missing profile", func(t *testing.T) {
		_, err := factory(context.Background(), "nonexistent", "us-east-1")
		if err == nil {
			t.Fatal("expected error for missing profile, got nil")
		}
	})
}

func TestDiscoverEKSClustersWithAWSVault(t *testing.T) {
	tmpDir := t.TempDir()
	configData, err := readFixture("config_valid")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	configPath := filepath.Join(tmpDir, "config")
	if err := writeFixture(configPath, string(configData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	// Set up mock aws-vault.
	mockScript := filepath.Join(tmpDir, "mock-aws-vault")
	creds := credentialProcessOutput{
		Version:         1,
		AccessKeyID:     "AKIATEST",
		SecretAccessKey: "SECRET",
		SessionToken:    "TOKEN",
		Expiration:      "2099-01-01T00:00:00Z",
	}
	jsonBytes, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	scriptContent := "#!/bin/sh\necho '" + string(jsonBytes) + "'\n"
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	old := awsVaultCommand
	awsVaultCommand = mockScript
	defer func() { awsVaultCommand = old }()

	cfg := DefaultConfig()
	cfg.AWS.ConfigFile = configPath
	cfg.AWS.CredentialsFile = ""
	cfg.AWS.Regions = []string{"us-east-1"}
	cfg.AWS.ParallelWorkers = 2
	cfg.AWS.Timeout = 100 * time.Millisecond

	// Use a mock EKS client factory that wraps the aws-vault credentials.
	// DiscoverEKSClusters will detect aws-vault and create its own factory,
	// but we pass a custom factory to test the integration with pre-fetched creds.
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

	expected := []DiscoveredCluster{
		{
			Profile:  "prod",
			Region:   "us-east-1",
			Name:     "prod-us-east-1",
			Endpoint: "https://prod-us-east-1.example.com",
			CAData:   []byte("demo"),
		},
		{
			Profile:  "staging",
			Region:   "us-east-1",
			Name:     "staging-us-east-1",
			Endpoint: "https://staging-us-east-1.example.com",
			CAData:   []byte("demo"),
		},
	}

	if !reflect.DeepEqual(clusters, expected) {
		t.Fatalf("clusters = %#v", clusters)
	}
}
