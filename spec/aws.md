# AWS Provider Specification

## Overview

The AWS provider generates `~/.aws/config` profiles by discovering accounts and roles from AWS IAM Identity Center (SSO). It uses the AWS SSO APIs directly to enumerate available accounts and permission sets, then generates profiles with configurable naming templates.

## Goals

- Auto-discover all accounts/roles available via AWS SSO
- Generate `~/.aws/config` with configurable profile naming
- Support role filtering during discovery (by role name)
- Support role assumption chains (cross-account roles)
- Optional `~/.aws/credentials` generation with `credential_process` entries
- Backup existing config before overwriting
- Support shared SSO session sections
- Support pruning stale generated profiles

## Non-Goals

- SSO login handling (users run `aws sso login` or `granted login` separately)
- GKE/Azure support
- Managing SSO sessions or token caching beyond selecting an existing token

## Features

### SSO Profile Discovery

1. Authenticate with AWS SSO using cached tokens (from AWS CLI and Granted caches)
2. Select the newest valid token across all cache locations
3. Call `ListAccounts` API to enumerate accessible accounts
4. For each account, call `ListAccountRoles` to get available permission sets
5. Filter results by role name if `--role` or `roles` config is provided
6. Generate profiles using configurable template and optional prefix

### Shared SSO Sessions

The provider writes a shared SSO session section and references it in each generated profile:

```ini
[sso-session lazycfg]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access

[profile prod-account/AdminAccess]
sso_session = lazycfg
sso_account_id = 123456789012
sso_role_name = AdminAccess
```

### Profile Generation

Each discovered account/role combination becomes a profile in `~/.aws/config`:

```ini
[profile prod-account/AdminAccess]
sso_session = lazycfg
sso_account_id = 123456789012
sso_role_name = AdminAccess
```

When `use_credential_process: true`:

```ini
[profile prod-account/AdminAccess]
credential_process = granted credential-process --profile prod-account/AdminAccess
```

### Role Chains

Define cross-account role assumptions in config:

```yaml
providers:
  aws:
    role_chains:
      - name: prod-readonly
        source_profile: prod-account/AdminAccess
        role_arn: arn:aws:iam::111111111111:role/ReadOnly
      - name: staging-deploy
        source_profile: staging-account/PowerUser
        role_arn: arn:aws:iam::222222222222:role/DeployRole
```

Generated output:

```ini
[profile prod-readonly]
source_profile = prod-account/AdminAccess
role_arn = arn:aws:iam::111111111111:role/ReadOnly

[profile staging-deploy]
source_profile = staging-account/PowerUser
role_arn = arn:aws:iam::222222222222:role/DeployRole
```

### Prune Stale Profiles

When `prune: true`, the provider removes previously generated profiles that are no longer present in the latest discovery. Pruning only applies to profiles marked with the marker key (default: `automatically_generated`).

### Backup Strategy

Before writing to `~/.aws/config`:

1. Check if file exists
2. If exists, create backup: `~/.aws/config.bak.YYYYMMDD-HHMMSS`
3. Write new config

## Configuration

```yaml
providers:
  aws:
    enabled: true

    # SSO configuration
    sso:
      start_url: https://example.awsapps.com/start
      region: us-east-1
      session_name: lazycfg

    # Output paths
    config_path: ~/.aws/config
    credentials_path: ~/.aws/credentials # only used if generate_credentials: true

    # Profile generation
    profile_template: "{{ .AccountName }}/{{ .RoleName }}"
    profile_prefix: ""
    marker_key: automatically_generated
    prune: false
    generate_credentials: false # if true, also write credentials file
    use_credential_process: false # if true, use granted credential-process

    # Role filtering (empty = all roles)
    roles: [] # e.g., ["AdminAccess", "PowerUser"]

    # Role chains for cross-account access
    role_chains: []

    # Backup settings
    backup_enabled: true
    backup_dir: ~/.aws # where to store .bak files

    # Token cache locations (searched in order, newest valid token wins)
    token_cache_paths:
      - ~/.aws/sso/cache
      - ~/.granted/sso
```

### Profile Template Variables

Templates use Go text/template syntax. Available variables:

| Variable             | Alias                                   | Description               |
| -------------------- | --------------------------------------- | ------------------------- |
| `{{ .AccountName }}` | `{{ .account_name }}`, `{{ .account }}` | AWS account name from SSO |
| `{{ .AccountID }}`   | `{{ .account_id }}`                     | 12-digit AWS account ID   |
| `{{ .RoleName }}`    | `{{ .role_name }}`, `{{ .role }}`       | Permission set name       |
| `{{ .SSORegion }}`   | `{{ .sso_region }}`                     | SSO region                |

Example templates:

- `{{ .AccountName }}/{{ .RoleName }}` → `prod-account/AdminAccess`
- `{{ .account }}-{{ .role }}` → `prod-account-AdminAccess`
- `{{ .AccountID }}-{{ .RoleName }}` → `123456789012-AdminAccess`

## CLI Interface

```bash
# Generate AWS config from SSO
lazycfg generate aws

# Filter to specific roles
lazycfg generate aws --role AdminAccess
lazycfg generate aws --role AdminAccess --role PowerUser

# Use credential_process instead of native SSO fields
lazycfg generate aws --credential-process

# Custom profile template
lazycfg generate aws --template "{{ .account }}-{{ .role }}"

# Prefix all generated profiles
lazycfg generate aws --prefix sso_

# Prune stale profiles
lazycfg generate aws --prune

# Also generate credentials file
lazycfg generate aws --credentials

# Dry run
lazycfg generate aws --dry-run

# Force overwrite (no backup prompt)
lazycfg generate aws --force

# Demo mode (fake data, no AWS calls)
lazycfg generate aws --demo
```

## Implementation

### Directory Structure

```
internal/providers/aws/
  client.go           # AWS SSO API client interface
  client_mock.go      # Mock client for testing
  config.go           # Config struct, validation, defaults
  discovery.go        # SSO account/role discovery logic
  generator.go        # Profile generation and template handling
  provider.go         # Provider implementation
  provider_test.go    # Tests
  testdata/           # Fixtures
```

### Core Types

```go
// Config represents AWS provider configuration.
type Config struct {
    BackupDir             string       `yaml:"backup_dir"`
    BackupEnabled         bool         `yaml:"backup_enabled"`
    ConfigPath            string       `yaml:"config_path"`
    CredentialsPath       string       `yaml:"credentials_path"`
    Enabled               bool         `yaml:"enabled"`
    GenerateCredentials   bool         `yaml:"generate_credentials"`
    MarkerKey             string       `yaml:"marker_key"`
    ProfilePrefix         string       `yaml:"profile_prefix"`
    ProfileTemplate       string       `yaml:"profile_template"`
    Prune                 bool         `yaml:"prune"`
    Roles                 []string     `yaml:"roles"`
    RoleChains            []RoleChain  `yaml:"role_chains"`
    SSO                   SSOConfig    `yaml:"sso"`
    TokenCachePaths       []string     `yaml:"token_cache_paths"`
    UseCredentialProcess  bool         `yaml:"use_credential_process"`
}

type SSOConfig struct {
    Region      string `yaml:"region"`
    SessionName string `yaml:"session_name"`
    StartURL    string `yaml:"start_url"`
}

type RoleChain struct {
    Name          string `yaml:"name"`
    RoleARN       string `yaml:"role_arn"`
    SourceProfile string `yaml:"source_profile"`
    Region        string `yaml:"region,omitempty"`  // optional
}

// DiscoveredProfile represents an account/role from SSO.
type DiscoveredProfile struct {
    AccountID   string
    AccountName string
    RoleName    string
    SSOStartURL string
    SSORegion   string
}
```

### AWS SSO Client Interface

```go
// SSOClient defines the interface for AWS SSO operations.
type SSOClient interface {
    ListAccounts(ctx context.Context, params *sso.ListAccountsInput) (*sso.ListAccountsOutput, error)
    ListAccountRoles(ctx context.Context, params *sso.ListAccountRolesInput) (*sso.ListAccountRolesOutput, error)
}

// SSOClientFactory creates SSO clients with the appropriate credentials.
type SSOClientFactory func(ctx context.Context, region string) (SSOClient, error)
```

### Discovery Flow

1. Load SSO access tokens from `token_cache_paths`
2. Select the newest valid token across all caches
3. If no valid token found, return error: "SSO session expired. Run 'aws sso login' first."
4. Create SSO client with token
5. Paginate through `ListAccounts`
6. For each account, paginate through `ListAccountRoles`
7. Filter by role name if configured
8. Return slice of `DiscoveredProfile`

### Profile Generation Flow

1. Parse profile template
2. For each discovered profile:
   - Execute template to generate profile name
   - Apply `profile_prefix`
   - Build profile section (SSO fields or credential_process)
   - Add marker key to generated profiles
3. For each role chain:
   - Build profile section with source_profile and role_arn
4. Format as INI content
5. Overwrite profiles on collision

### Prune Flow

1. Load existing config file
2. Identify profiles with the marker key
3. Remove marked profiles not present in the newly generated set
4. Preserve all unmarked profiles

### Backup Flow

```go
func backupConfig(configPath string) (string, error) {
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return "", nil
    }

    timestamp := time.Now().Format("20060102-150405")
    backupPath := fmt.Sprintf("%s.bak.%s", configPath, timestamp)

    if err := copyFile(configPath, backupPath); err != nil {
        return "", fmt.Errorf("backup config: %w", err)
    }

    return backupPath, nil
}
```

## Dependencies

```go
require (
    github.com/aws/aws-sdk-go-v2/config
    github.com/aws/aws-sdk-go-v2/service/sso
    gopkg.in/ini.v1  // for INI file generation
)
```

## Testing

- Mock `SSOClient` interface for unit tests
- Use `t.TempDir()` for file operations
- Test fixtures: sample SSO cache tokens, existing config files
- `--demo` flag injects mock client with fake accounts/roles
- Test cases:
  - Discovery with valid token
  - Discovery with expired token (error)
  - Token selection across multiple caches
  - Role filtering
  - Template expansion and prefix
  - Profile overwrite on collision
  - Role chain generation
  - Marker key injection
  - Prune stale profiles
  - Backup creation
  - Credentials file generation
  - Dry run mode

## Error Handling

| Scenario                              | Behavior                                                                       |
| ------------------------------------- | ------------------------------------------------------------------------------ |
| No SSO token found                    | Error: "No SSO session found. Run 'aws sso login --sso-session <name>' first." |
| SSO token expired                     | Error: "SSO session expired. Run 'aws sso login' to refresh."                  |
| Invalid template                      | Error: "Invalid profile template: <parse error>"                               |
| Missing start_url                     | Error: "sso.start_url is required"                                             |
| Missing region                        | Error: "sso.region is required"                                                |
| Role chain references unknown profile | Warning: "source_profile 'X' not found in discovered profiles"                 |

## Decisions

1. **Direct API vs Shell Out**: Use AWS SSO APIs directly for full control and no external dependencies.
2. **Single SSO Instance**: One start_url per config; run multiple times for multiple SSO instances.
3. **No Auto-Login**: Prompt user to login manually; keeps provider stateless and simple.
4. **Backup Always**: Always backup existing config unless `--force` or `no_backup: true`.
5. **Native SSO Fields by Default**: Use `sso_*` fields; `credential_process` is opt-in.
6. **Filter During Discovery**: Apply role filter early to reduce API calls.
7. **Shared SSO Sessions**: Generate `[sso-session <name>]` and reference via `sso_session`.
8. **Prefix and Prune**: Support profile prefixes and prune with marker key.
9. **Token Selection**: Use newest valid token from AWS CLI or Granted caches.
10. **Collision Handling**: Overwrite profiles on name collision.

## Implementation Phases

### Phase 1: Core Discovery

- [ ] SSO client interface and mock
- [ ] Token loading from cache
- [ ] Account/role discovery with filtering
- [ ] Basic profile generation (native SSO fields)
- [ ] Unit tests

### Phase 2: Config Generation

- [ ] Profile template parsing and expansion
- [ ] INI file generation
- [ ] Backup mechanism
- [ ] Shared SSO session generation
- [ ] Config file writing
- [ ] CLI integration

### Phase 3: Extended Features

- [ ] Role chain support
- [ ] Credentials file generation (credential_process)
- [ ] Prefix support
- [ ] Prune support
- [ ] Demo mode
- [ ] Documentation
