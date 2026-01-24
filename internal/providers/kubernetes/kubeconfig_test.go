package kubernetes

import (
	"reflect"
	"testing"
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
