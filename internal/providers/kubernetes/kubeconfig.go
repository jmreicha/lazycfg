package kubernetes

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/client-go/tools/clientcmd/api"
)

var errNamingPatternInvalid = errors.New("naming pattern contains unknown placeholders")

// BuildKubeconfig constructs a kubeconfig from discovered clusters.
func BuildKubeconfig(clusters []DiscoveredCluster, namingPattern string) (*api.Config, error) {
	namingPattern = strings.TrimSpace(namingPattern)
	if namingPattern == "" {
		return nil, errNamingPatternEmpty
	}

	config := &api.Config{
		Kind:           kubeconfigKind,
		APIVersion:     "v1",
		Clusters:       make(map[string]*api.Cluster),
		AuthInfos:      make(map[string]*api.AuthInfo),
		Contexts:       make(map[string]*api.Context),
		Preferences:    api.Preferences{},
		CurrentContext: "",
	}

	for _, cluster := range clusters {
		name, err := renderNamingPattern(namingPattern, cluster)
		if err != nil {
			return nil, err
		}

		config.Clusters[name] = &api.Cluster{
			Server:                   cluster.Endpoint,
			CertificateAuthorityData: cluster.CAData,
		}

		config.AuthInfos[name] = &api.AuthInfo{
			Exec: execConfigForCluster(cluster),
		}

		config.Contexts[name] = &api.Context{
			Cluster:  name,
			AuthInfo: name,
		}
	}

	return config, nil
}

func execConfigForCluster(cluster DiscoveredCluster) *api.ExecConfig {
	if cluster.AuthMode == "aws-vault" {
		return &api.ExecConfig{
			APIVersion: "client.authentication.k8s.io/v1",
			Command:    "aws-vault",
			Args: []string{
				"exec",
				cluster.Profile,
				"--",
				"aws",
				"eks",
				"get-token",
				"--cluster-name",
				cluster.Name,
				"--region",
				cluster.Region,
			},
			Env: []api.ExecEnvVar{
				{
					Name:  "AWS_PROFILE",
					Value: accountFromProfile(cluster.Profile),
				},
			},
			InteractiveMode:    api.IfAvailableExecInteractiveMode,
			ProvideClusterInfo: false,
		}
	}

	return &api.ExecConfig{
		APIVersion: "client.authentication.k8s.io/v1",
		Command:    "aws",
		Args: []string{
			"eks",
			"get-token",
			"--cluster-name",
			cluster.Name,
			"--region",
			cluster.Region,
		},
		Env: []api.ExecEnvVar{
			{
				Name:  "AWS_PROFILE",
				Value: cluster.Profile,
			},
		},
	}
}

func renderNamingPattern(pattern string, cluster DiscoveredCluster) (string, error) {
	replacements := map[string]string{
		"{cluster}": cluster.Name,
		"{profile}": cluster.Profile,
		"{region}":  cluster.Region,
	}

	rendered := pattern
	for token, value := range replacements {
		rendered = strings.ReplaceAll(rendered, token, value)
	}

	if strings.Contains(rendered, "{") || strings.Contains(rendered, "}") {
		return "", fmt.Errorf("%w: %s", errNamingPatternInvalid, pattern)
	}

	rendered = strings.TrimSpace(rendered)
	if rendered == "" {
		return "", errors.New("naming pattern resolved to empty name")
	}

	return rendered, nil
}

// accountFromProfile extracts the account portion of a profile name.
// For "account/role" profiles it returns "account"; otherwise the full profile.
func accountFromProfile(profile string) string {
	if idx := strings.Index(profile, "/"); idx >= 0 {
		return profile[:idx]
	}
	return profile
}
