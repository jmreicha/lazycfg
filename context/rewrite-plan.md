# Lazycfg Rewrite Plan

## Executive Summary

Complete rewrite of lazycfg with a plugin-based modular architecture. The goal is to create an extensible configuration management tool that can handle multiple config types (AWS/Granted, Kubernetes, SSH) with a CLI-first interface and optional YAML configuration support.

## Why Rewrite?

- **Architecture limitations**: Current structure doesn't support extensibility
- **Code quality**: Technical debt makes maintenance and extension difficult
- **Scope evolution**: Vision has expanded beyond original implementation
- **Fresh start**: Clean slate allows applying all lessons learned

## Core Principles

1. **Plugin-based architecture**: Each config type is an isolated, swappable provider
2. **CLI-first**: Primary interaction via command-line flags with optional YAML config
3. **Native Go**: Use standard library and native Go libraries over external commands
4. **Extensibility**: Easy to add new configuration providers
5. **Type safety**: Leverage Go's type system for validation and safety
6. **Testability**: Design for comprehensive test coverage

## Architecture Overview

### High-Level Design

```
┌─────────────────────────────────────────────────────────┐
│                      CLI Layer                           │
│  (cobra/kong + flag parsing + YAML config override)     │
└─────────────────┬───────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│                   Core Engine                            │
│  - Provider registry                                     │
│  - Lifecycle management                                  │
│  - Validation orchestration                              │
│  - Backup/rollback                                       │
└─────────────────┬───────────────────────────────────────┘
                  │
        ┌─────────┴─────────┬───────────────┐
        │                   │               │
┌───────▼────────┐  ┌───────▼────────┐  ┌──▼──────────┐
│ AWS Provider   │  │  K8s Provider  │  │SSH Provider │
│ - Granted cfg  │  │ - kubeconfig   │  │- ssh config │
│ - AWS profiles │  │ - EKS clusters │  │- known hosts│
│ - Steampipe    │  │ - contexts     │  │             │
└────────────────┘  └────────────────┘  └─────────────┘
```

### Core Interfaces

#### Provider Interface

```go
type Provider interface {
    // Name returns the provider identifier
    Name() string

    // Validate checks if prerequisites are met
    Validate(ctx context.Context) error

    // Generate creates the configuration
    Generate(ctx context.Context, opts GenerateOptions) (*Result, error)

    // Backup creates a backup of existing configuration
    Backup(ctx context.Context) (string, error)

    // Restore recovers from a backup
    Restore(ctx context.Context, backupPath string) error

    // Clean removes generated configuration
    Clean(ctx context.Context) error
}
```

#### Configuration Model

```go
type Config struct {
    // Global settings
    Verbose bool
    DryRun  bool
    Backup  bool

    // Provider-specific configs
    Providers map[string]ProviderConfig
}

type ProviderConfig interface {
    Validate() error
}
```

### Directory Structure

```
lazycfg/
├── cmd/
│   └── lazycfg/
│       └── main.go                 # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                 # Root command
│   │   ├── generate.go             # Generate commands
│   │   ├── clean.go                # Clean commands
│   │   └── version.go              # Version command
│   ├── core/
│   │   ├── provider.go             # Provider interface
│   │   ├── registry.go             # Provider registry
│   │   ├── engine.go               # Core engine
│   │   ├── config.go               # Configuration model
│   │   └── backup.go               # Backup/restore logic
│   ├── providers/
│   │   ├── aws/
│   │   │   ├── provider.go         # AWS provider implementation
│   │   │   ├── granted.go          # Granted config generation
│   │   │   ├── steampipe.go        # Steampipe config generation
│   │   │   └── config.go           # AWS-specific config
│   │   ├── kubernetes/
│   │   │   ├── provider.go         # K8s provider implementation
│   │   │   ├── kubeconfig.go       # Kubeconfig management
│   │   │   ├── eks.go              # EKS integration
│   │   │   └── config.go           # K8s-specific config
│   │   └── ssh/
│   │       ├── provider.go         # SSH provider implementation
│   │       ├── sshconfig.go        # SSH config management
│   │       └── config.go           # SSH-specific config
│   └── util/
│       ├── file.go                 # File operations
│       ├── template.go             # Template rendering
│       └── command.go              # Command execution
├── pkg/
│   └── lazycfg/
│       └── client.go               # Public API (future SDK)
├── config/
│   └── lazycfg.example.yaml        # Example configuration
└── README.md
```

## Implementation Roadmap

### Phase 0: Foundation (Week 1)

**Goal**: Establish core architecture and interfaces

- [ ] Define core interfaces (Provider, Config, Registry)
- [ ] Implement provider registry
- [ ] Create core engine with lifecycle management
- [ ] Set up CLI framework (cobra or kong)
- [ ] Implement config loading (flags + YAML)
- [ ] Add backup/restore functionality
- [ ] Write comprehensive tests for core

### Phase 1: AWS Provider (Week 2)

**Goal**: Replicate and improve existing AWS/Granted functionality

- [ ] Implement AWS provider skeleton
- [ ] Port Granted config generation
- [ ] Add Steampipe config generation
- [ ] Implement AWS profile discovery
- [ ] Add SSO configuration support
- [ ] Create AWS-specific CLI commands
- [ ] Write provider tests
- [ ] Add E2E tests for AWS workflows

### Phase 2: Kubernetes Provider (Week 3)

**Goal**: Implement comprehensive K8s config management

- [ ] Implement K8s provider skeleton
- [ ] Build kubeconfig parser
- [ ] Implement kubeconfig merging logic
- [ ] Add EKS cluster discovery
- [ ] Integrate with AWS provider for EKS auth
- [ ] Add context management
- [ ] Support secure auth with aws-vault
- [ ] Write provider tests
- [ ] Add E2E tests for K8s workflows

### Phase 3: SSH Provider (Week 4)

**Goal**: Add SSH configuration management

- [ ] Implement SSH provider skeleton
- [ ] Build SSH config parser
- [ ] Implement config generation
- [ ] Add known_hosts management
- [ ] Support SSH key management
- [ ] Write provider tests
- [ ] Add E2E tests for SSH workflows

### Phase 4: Polish & Extensions (Week 5)

**Goal**: Production readiness and developer experience

- [ ] Implement self-update functionality
- [ ] Add shell completions
- [ ] Create documentation site
- [ ] Add verbose/debug logging
- [ ] Improve error messages
- [ ] Add dry-run support across all providers
- [ ] Performance optimization
- [ ] CI/CD pipeline setup

### Phase 5: Distribution (Week 6)

**Goal**: Make it easy to install and use

- [ ] Create installation script
- [ ] Set up Homebrew tap
- [ ] Build Docker image
- [ ] Create deb packages
- [ ] Add goreleaser configuration
- [ ] Write installation documentation

## Configuration Examples

### CLI-First Approach

```bash
# Generate AWS/Granted config
lazycfg generate aws --sso-start-url https://example.awsapps.com/start

# Generate all configs
lazycfg generate all --config lazycfg.yaml

# Clean specific provider
lazycfg clean aws

# Dry run
lazycfg generate kubernetes --dry-run
```

### YAML Configuration (Optional)

```yaml
# lazycfg.yaml
backup: true
verbose: false

providers:
  aws:
    enabled: true
    sso_start_url: https://example.awsapps.com/start
    sso_region: us-west-2
    credential_process_auto_login: true
    steampipe:
      enabled: true
      plugins:
        - aws
        - cloudflare
        - kubernetes

  kubernetes:
    enabled: true
    merge_configs: true
    use_aws_vault: false

  ssh:
    enabled: true
    backup_interval: 7d
```

## Technical Decisions

### CLI Framework

**Decision**: Use Kong (current choice) or consider Cobra

**Rationale**:

- Kong: Simpler, tag-based, less boilerplate (current)
- Cobra: More features, widely adopted, better docs

**Recommendation**: Stick with Kong for simplicity unless Cobra features needed

### Configuration Loading

**Priority**: CLI flags > YAML config > defaults

```go
// Example precedence
func LoadConfig() (*Config, error) {
    cfg := DefaultConfig()

    // Load YAML if exists
    if yamlCfg, err := loadYAML(); err == nil {
        cfg.Merge(yamlCfg)
    }

    // Override with CLI flags
    cfg.ApplyFlags(cliFlags)

    return cfg, nil
}
```

### Dependency Management

- Use Go modules for dependency management
- Prefer standard library over external deps
- Use well-maintained libraries for specific needs:
  - CLI: kong (or cobra)
  - AWS: aws-sdk-go-v2
  - K8s: client-go
  - YAML: gopkg.in/yaml.v3

### Testing Strategy

1. **Unit tests**: All core logic and providers
2. **Integration tests**: Provider interactions with real configs
3. **E2E tests**: Full workflows using test fixtures
4. **Table-driven tests**: For validation and parsing logic

## Migration Strategy

Since this is a complete rewrite:

1. **No backward compatibility required**: Fresh start
2. **Feature parity first**: Match existing functionality
3. **Additive improvements**: Add new features after parity
4. **Documentation**: Clear migration guide for any existing users

## Success Criteria

- [ ] All existing functionality replicated
- [ ] Plugin architecture in place and tested
- [ ] 3 providers implemented (AWS, K8s, SSH)
- [ ] Test coverage >80%
- [ ] Documentation complete
- [ ] Installation packages available
- [ ] CI/CD pipeline operational

## Next Steps

1. Review and approve this plan
2. Set up project tracking (GitHub issues or beads)
3. Start Phase 0 implementation
4. Regular check-ins at phase boundaries

## Questions to Resolve

1. Should we use Cobra instead of Kong for more features?
2. What's the preferred AWS SDK integration approach?
3. Should providers be compiled in or loaded dynamically?
4. Do we need a plugin API for external providers?
5. What's the minimum Go version target? (suggest 1.23+)
