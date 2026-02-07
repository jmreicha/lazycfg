# Kubernetes Provider Specification

## Overview

The Kubernetes provider discovers EKS clusters across AWS profiles and generates a merged kubeconfig file. It supports parallel cluster discovery and optional merging of existing kubeconfig files.

## Features

### EKS Cluster Discovery

1. Parse `~/.aws/credentials` to extract profile names
2. For each profile/region combination (in parallel):
   - List EKS clusters
   - Call `DescribeCluster` to get endpoint and CA data
3. Generate merged kubeconfig with all discovered clusters

### Kubeconfig Merge (optional)

When `--merge` flag is passed, also merge existing kubeconfig files from `~/.kube/` into the output.

Merge operates independently of discovery. Can use `--merge` alone to combine existing files without AWS calls.

## Configuration

```yaml
providers:
  kubernetes:
    enabled: true
    config_path: ~/.kube/config # output file path

    aws:
      credentials_file: ~/.aws/credentials
      profiles: [] # empty = all profiles
      regions: # default regions to scan
        - us-east-1
        - us-west-2
        - eu-west-1
      parallel_workers: 10
      timeout: 30s

    naming_pattern: "{profile}-{cluster}" # context/cluster/user naming

    merge:
      source_dir: ~/.kube
      include_patterns: ["*.yaml", "*.yml", "config"]
      exclude_patterns: ["*.bak", "*.backup"]
```

## CLI Interface

```bash
# Generate kubeconfig for all EKS clusters
lazycfg generate kubernetes

# Generate with specific profiles/regions
lazycfg generate kubernetes --profiles prod,staging
lazycfg generate kubernetes --regions us-east-1,us-west-2

# Merge existing configs only (no AWS discovery)
lazycfg generate kubernetes --merge-only

# Discover clusters AND merge existing configs
lazycfg generate kubernetes --merge

# Dry run
lazycfg generate kubernetes --dry-run

# Demo mode - use fake data, no AWS calls (for testing/development)
lazycfg generate kubernetes --demo
```

## Implementation

### Directory Structure

```
internal/providers/kubernetes/
  config.go         # Config struct, validation, defaults
  provider.go       # Provider implementation + discovery + merge
  provider_test.go  # tests
  testdata/         # fixtures
```

### Core Types

```go
type Config struct {
    Enabled       bool        `yaml:"enabled"`
    ConfigPath    string      `yaml:"config_path"`
    AWS           AWSConfig   `yaml:"aws"`
    NamingPattern string      `yaml:"naming_pattern"`
    Merge         MergeConfig `yaml:"merge"`
}

type AWSConfig struct {
    CredentialsFile string        `yaml:"credentials_file"`
    Profiles        []string      `yaml:"profiles"`
    Regions         []string      `yaml:"regions"`
    ParallelWorkers int           `yaml:"parallel_workers"`
    Timeout         time.Duration `yaml:"timeout"`
}

type MergeConfig struct {
    SourceDir       string   `yaml:"source_dir"`
    IncludePatterns []string `yaml:"include_patterns"`
    ExcludePatterns []string `yaml:"exclude_patterns"`
}

type DiscoveredCluster struct {
    Profile  string
    Region   string
    Name     string
    Endpoint string
    CAData   []byte
}
```

### AWS Client Interface

```go
type EKSClient interface {
    ListClusters(ctx context.Context, params *eks.ListClustersInput, optFns ...func(*eks.Options)) (*eks.ListClustersOutput, error)
    DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
}

type EKSClientFactory func(ctx context.Context, profile, region string) (EKSClient, error)
```

### Kubeconfig Generation

Uses `k8s.io/client-go/tools/clientcmd/api` types. Each discovered cluster becomes:

- Cluster entry: `{name, endpoint, ca-data}`
- AuthInfo entry: `{exec: aws eks get-token --cluster-name X --region Y}` with `AWS_PROFILE` env
- Context entry: references cluster and authinfo

### Merge Strategy

Overwrite on conflict. When merging:

1. Start with existing config (if present) or empty config
2. Merge each source file (clusters, contexts, authinfos)
3. Add discovered EKS clusters (overwrites if name collision)
4. Write to output path

## Dependencies

```go
require (
    github.com/aws/aws-sdk-go-v2/config
    github.com/aws/aws-sdk-go-v2/service/eks
    k8s.io/client-go
)
```

Required tools:

- `kubectl`
- `k9s`

## Testing

- Mock `EKSClient` interface for unit tests
- Use `t.TempDir()` for file operations
- Test fixtures: sample credentials files, kubeconfig files
- `--demo` flag injects mock factory with fake clusters for end-to-end CLI testing without AWS

## Decisions

1. **GKE/Azure**: Not in scope
2. **Output mode**: Merged only (no individual files)
3. **Conflict strategy**: Overwrite (simplest, most practical)
4. **Auth handling**: Out of scope, focus on config generation
5. **Default regions**: Sensible subset, not all ~30 AWS regions
