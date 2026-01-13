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
│ SSH Provider   │  │  K8s Provider  │  │AWS Provider │
│ - ssh config   │  │ - kubeconfig   │  │- Granted cfg│
│                │  │ - contexts     │  │- Steampipe  │
│                │  │                │  │- AWS profiles│
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
│   │   │   └── config.go           # K8s-specific config
│   │   └── ssh/
│   │       ├── provider.go         # SSH provider implementation
│   │       ├── parser.go           # SSH config parser
│   │       ├── generator.go        # SSH config generator
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

### Implementation Strategy

**Why SSH Provider First?**

The implementation order has been adjusted to build the SSH provider before AWS and Kubernetes providers. This approach provides several key benefits:

1. **Architecture validation**: SSH config is a well-defined, text-based format - ideal for proving the provider plugin system works correctly
2. **Minimal dependencies**: No external SDKs or cloud service requirements, fewer moving parts, faster development cycle
3. **Clear scope**: Straightforward config structure compared to AWS multi-tool complexity (Granted, Steampipe, profiles, SSO)
4. **Independent testing**: No AWS accounts or credentials needed for development and testing
5. **Faster iteration**: Quick feedback on provider interface design decisions and patterns

This "simplest-first" approach allows us to validate and refine the core plugin architecture before tackling more complex providers.

### Phase 0: Foundation (Week 1)

**Goal**: Establish core architecture and interfaces

- [x] Define core interfaces (Provider, Config, Registry)
- [x] Implement provider registry
- [x] Create core engine with lifecycle management
- [x] Set up CLI framework (using Cobra)
- [x] Implement config loading (flags + YAML)
- [x] Add backup/restore functionality
- [x] Write comprehensive tests for core

**Status**: ✅ Complete

### Phase 1: SSH Provider (Week 2)

**Goal**: Validate provider architecture with simple, well-defined SSH config generation

- [ ] Implement SSH provider skeleton (`internal/providers/ssh/`)
- [ ] Build SSH config parser (handle Host blocks, directives, comments, Include statements)
- [ ] Implement config generation for `~/.ssh/config`
- [ ] Implement backup/restore functionality
- [ ] Wire SSH provider to CLI (register in registry, add flags)
- [ ] Implement SSH provider configuration struct with validation
- [ ] Write comprehensive tests (unit, integration, E2E)

**Scope**: Focus on SSH config generation only. Known_hosts and SSH key management are deferred to later phases.

### Phase 2: Kubernetes Provider (Week 3)

**Goal**: Implement basic kubeconfig management without cloud provider dependencies

- [ ] Implement K8s provider skeleton
- [ ] Build kubeconfig parser (YAML-based)
- [ ] Implement kubeconfig merging logic
- [ ] Add context management
- [ ] Write provider tests
- [ ] Add E2E tests for K8s workflows

**Scope**: Basic kubeconfig management only. EKS cluster discovery and AWS integration are deferred to Phase 3.

### Phase 3: AWS Provider (Week 4)

**Goal**: Implement complex AWS tooling with Granted, Steampipe, and cloud integrations

- [ ] Implement AWS provider skeleton
- [ ] Port Granted config generation (OS-specific templates)
- [ ] Add Steampipe config generation (aggregator connections)
- [ ] Implement AWS profile discovery
- [ ] Add SSO configuration support
- [ ] Create AWS-specific CLI commands
- [ ] Add EKS integration to K8s provider
- [ ] Write provider tests
- [ ] Add E2E tests for AWS workflows

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
# Generate SSH config
lazycfg generate ssh

# Generate AWS/Granted config
lazycfg generate aws --sso-start-url https://example.awsapps.com/start

# Generate all configs
lazycfg generate all --config lazycfg.yaml

# Clean specific provider
lazycfg clean ssh

# Dry run
lazycfg generate kubernetes --dry-run
```

### YAML Configuration (Optional)

```yaml
# lazycfg.yaml
backup: true
verbose: false

providers:
  ssh:
    enabled: true
    hosts:
      - name: clab
        hostname: clab
        user: jmreicha
      - name: sulaco
        hostname: sulaco
        user: jmreicha

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
- [ ] 3 providers implemented (SSH, K8s, AWS)
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
