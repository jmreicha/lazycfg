package kubernetes

import (
	"reflect"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"
)

func TestBuildKubeconfig(t *testing.T) {
	clusters := []DiscoveredCluster{
		{
			Profile:  "prod",
			Region:   "us-west-2",
			Name:     "payments",
			Endpoint: "https://payments.example.com",
			CAData:   []byte("ca-data"),
		},
	}

	cfg, err := BuildKubeconfig(clusters, "{profile}-{cluster}")
	if err != nil {
		t.Fatalf("BuildKubeconfig failed: %v", err)
	}

	if len(cfg.Clusters) != 1 {
		t.Fatalf("clusters = %d", len(cfg.Clusters))
	}

	name := "prod-payments"
	cluster := cfg.Clusters[name]
	if cluster == nil {
		t.Fatalf("cluster %q not found", name)
	}

	if cluster.Server != "https://payments.example.com" {
		t.Errorf("cluster.Server = %q", cluster.Server)
	}

	authInfo := cfg.AuthInfos[name]
	if authInfo == nil || authInfo.Exec == nil {
		t.Fatalf("auth info missing for %q", name)
	}

	if authInfo.Exec.Command != "aws" {
		t.Errorf("Exec.Command = %q", authInfo.Exec.Command)
	}

	expectedArgs := []string{"eks", "get-token", "--cluster-name", "payments", "--region", "us-west-2"}
	if !reflect.DeepEqual(authInfo.Exec.Args, expectedArgs) {
		t.Errorf("Exec.Args = %v", authInfo.Exec.Args)
	}

	if len(authInfo.Exec.Env) != 1 || authInfo.Exec.Env[0].Value != "prod" {
		t.Errorf("Exec.Env = %v", authInfo.Exec.Env)
	}
}

func TestBuildKubeconfigAWSVault(t *testing.T) {
	clusters := []DiscoveredCluster{
		{
			Profile:  "prod",
			Region:   "us-west-2",
			Name:     "payments",
			Endpoint: "https://payments.example.com",
			CAData:   []byte("ca-data"),
			AuthMode: "aws-vault",
		},
	}

	cfg, err := BuildKubeconfig(clusters, "{profile}-{cluster}")
	if err != nil {
		t.Fatalf("BuildKubeconfig failed: %v", err)
	}

	name := "prod-payments"
	authInfo := cfg.AuthInfos[name]
	if authInfo == nil || authInfo.Exec == nil {
		t.Fatalf("auth info missing for %q", name)
	}

	if authInfo.Exec.Command != "aws-vault" {
		t.Errorf("Exec.Command = %q, want %q", authInfo.Exec.Command, "aws-vault")
	}

	expectedArgs := []string{"exec", "prod", "--", "aws", "eks", "get-token", "--cluster-name", "payments", "--region", "us-west-2"}
	if !reflect.DeepEqual(authInfo.Exec.Args, expectedArgs) {
		t.Errorf("Exec.Args = %v, want %v", authInfo.Exec.Args, expectedArgs)
	}

	if len(authInfo.Exec.Env) != 1 || authInfo.Exec.Env[0].Name != "AWS_PROFILE" || authInfo.Exec.Env[0].Value != "prod" {
		t.Errorf("Exec.Env = %v, want [{AWS_PROFILE prod}]", authInfo.Exec.Env)
	}
}

func TestBuildKubeconfigAWSVaultProfileWithRole(t *testing.T) {
	clusters := []DiscoveredCluster{
		{
			Profile:  "surf-stg/cloudinfra",
			Region:   "us-west-2",
			Name:     "app",
			Endpoint: "https://app.example.com",
			CAData:   []byte("ca-data"),
			AuthMode: "aws-vault",
		},
	}

	cfg, err := BuildKubeconfig(clusters, "{profile}-{cluster}")
	if err != nil {
		t.Fatalf("BuildKubeconfig failed: %v", err)
	}

	authInfo := cfg.AuthInfos["surf-stg/cloudinfra-app"]
	if authInfo == nil || authInfo.Exec == nil {
		t.Fatal("auth info missing")
	}

	if authInfo.Exec.Env[0].Value != "surf-stg" {
		t.Errorf("AWS_PROFILE = %q, want %q", authInfo.Exec.Env[0].Value, "surf-stg")
	}

	if authInfo.Exec.InteractiveMode != api.IfAvailableExecInteractiveMode {
		t.Errorf("Exec.InteractiveMode = %q, want %q", authInfo.Exec.InteractiveMode, api.IfAvailableExecInteractiveMode)
	}

	if authInfo.Exec.ProvideClusterInfo {
		t.Error("Exec.ProvideClusterInfo = true, want false")
	}
}

func TestBuildKubeconfigErrors(t *testing.T) {
	_, err := BuildKubeconfig(nil, "")
	if err == nil {
		t.Fatal("expected error for empty naming pattern")
	}

	_, err = BuildKubeconfig([]DiscoveredCluster{{Name: "demo"}}, "{unknown}")
	if err == nil {
		t.Fatal("expected error for invalid naming pattern")
	}
}
