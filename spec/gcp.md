# GCP Provider Specification

## Overview

The GCP provider manages gcloud CLI configurations for handling multiple GCP accounts and projects. It discovers authenticated accounts, generates configuration profiles, and optionally manages Application Default Credentials (ADC) files for Go SDK usage.

## Goals

- Manage multiple gcloud CLI configurations (switch, create, list)
- Auto-discover authenticated GCP accounts from gcloud
- Generate gcloud configuration files with account/project/region settings
- Support Application Default Credentials (ADC) file generation for Go SDK
- Backup existing configurations before overwriting
- Support credential refresh detection

## Non-Goals

- GCP login handling (users run `gcloud auth login` separately)
- AWS/Azure support
- Direct GCP API calls for resource enumeration

## Features

### Account Discovery

1. List authenticated accounts via `gcloud auth list`
2. Parse output to extract account email and authenticated status
3. Detect currently active account
4. Show account type (user vs service account)

### Configuration Management

1. List existing gcloud configurations via `gcloud config configurations list`
2. Create new configurations with specified account and project
3. Activate/switch configurations
4. Delete configurations (except active)

### ADC File Generation

For Go SDK usage, generate `~/.config/gcloud/application_default_credentials.json`:

```json
{
  "type": "authorized_user",
  "client_id": "...",
  "client_secret": "...",
  "refresh_token": "..."
}
```

Or for service account:

```json
{
  "type": "service_account",
  "project_id": "...",
  "private_key_id": "...",
  "private_key": "...",
  "client_email": "...",
  "client_id": "...",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token"
}
```

### Backup Strategy

Before writing to ADC file:

1. Check if file exists
2. If exists, create backup: `~/.config/gcloud/application_default_credentials.json.bak.YYYYMMDD-HHMMSS`
3. Write new credentials

## Configuration

```yaml
providers:
  gcp:
    enabled: true

    # Account discovery
    discover_accounts: true

    # Configuration management
    configurations:
      - name: work
        account: user@company.com
        project: my-project
        region: us-east-1
        zone: us-east1-b
      - name: personal
        account: user@gmail.com
        project: personal-project
        region: us-central1

    # Default configuration to activate
    default_config: work

    # ADC file generation
    adc:
      enabled: true
      path: ~/.config/gcloud/application_default_credentials.json
      account: work # which config to use for ADC
      backup_enabled: true

    # Output paths
    config_path: ~/.config/gcloud # gcloud config directory
```

## CLI Interface

```bash
# Discover and list GCP accounts
cfgctl generate gcp

# List accounts only
cfgctl generate gcp --list-accounts

# List configurations
cfgctl generate gcp --list-configs

# Create and activate a configuration
cfgctl generate gcp --create-config my-config --account user@company.com --project my-project

# Switch to existing configuration
cfgctl generate gcp --activate work

# Generate ADC file from active config
cfgctl generate gcp --generate-adc

# Dry run
cfgctl generate gcp --dry-run

# Force overwrite (no backup prompt)
cfgctl generate gcp --force
```

## Implementation

### Directory Structure

```
internal/providers/gcp/
  client.go           # gcloud CLI wrapper interface
  client_mock.go      # Mock client for testing
  config.go           # Config struct, validation, defaults
  discovery.go        # Account discovery logic
  generator.go        # Config and ADC generation
  provider.go         # Provider implementation
  provider_test.go    # Tests
  testdata/           # Fixtures
```

### Core Types

```go
// Config represents GCP provider configuration.
type Config struct {
    Enabled         bool              `yaml:"enabled"`
    DiscoverAccounts bool            `yaml:"discover_accounts"`
    Configurations  []GCPConfig       `yaml:"configurations"`
    DefaultConfig   string            `yaml:"default_config"`
    ADC             ADCConfig         `yaml:"adc"`
    ConfigPath      string            `yaml:"config_path"`
}

type GCPConfig struct {
    Name    string `yaml:"name"`
    Account string `yaml:"account"`
    Project string `yaml:"project"`
    Region  string `yaml:"region"`
    Zone    string `yaml:"zone"`
}

type ADCConfig struct {
    Enabled        bool   `yaml:"enabled"`
    Path           string `yaml:"path"`
    Account        string `yaml:"account"`
    BackupEnabled  bool   `yaml:"backup_enabled"`
}

// DiscoveredAccount represents an authenticated GCP account.
type DiscoveredAccount struct {
    Account   string
    IsActive  bool
    Type      string  // "user" or "service_account"
}

// DiscoveredConfig represents a gcloud configuration.
type DiscoveredConfig struct {
    Name      string
    IsActive  bool
    Account   string
    Project   string
    Region    string
    Zone      string
}
```

### Gcloud Client Interface

```go
// GcloudClient defines the interface for gcloud CLI operations.
type GcloudClient interface {
    ListAccounts(ctx context.Context) ([]DiscoveredAccount, error)
    ListConfigurations(ctx context.Context) ([]DiscoveredConfig, error)
    CreateConfiguration(ctx context.Context, name string) error
    ActivateConfiguration(ctx context.Context, name string) error
    DeleteConfiguration(ctx context.Context, name string) error
    GetConfiguration(ctx context.Context, name string) (DiscoveredConfig, error)
    Run(ctx context.Context, args ...string) (string, error)
}

// GcloudClientFactory creates gcloud clients.
type GcloudClientFactory func() GcloudClient
```

### Account Discovery Flow

1. Run `gcloud auth list --format=json`
2. Parse JSON output to extract accounts
3. Mark the currently authenticated account as active
4. Return slice of `DiscoveredAccount`

### Configuration Management Flow

1. Run `gcloud config configurations list --format=json`
2. Parse JSON output to extract configurations
3. Return slice of `DiscoveredConfig`

### ADC Generation Flow

1. Get the specified configuration account
2. Run `gcloud auth application-default print-access-token` to get refresh token
3. Extract credentials from gcloud's ADC cache: `~/.config/gcloud/application_default_credentials.json`
4. Or use service account key if configured
5. Backup existing ADC file if enabled
6. Write new ADC file

### Backup Flow

```go
func backupADCFile(adcPath string) (string, error) {
    if _, err := os.Stat(adcPath); os.IsNotExist(err) {
        return "", nil
    }

    timestamp := time.Now().Format("20060102-150405")
    backupPath := fmt.Sprintf("%s.bak.%s", adcPath, timestamp)

    if err := copyFile(adcPath, backupPath); err != nil {
        return "", fmt.Errorf("backup ADC file: %w", err)
    }

    return backupPath, nil
}
```

## Dependencies

```go
require (
    golang.org/x/oauth2/google
    google.golang.org/api
)
```

Required tools:

- `gcloud`

## Testing

- Mock `GcloudClient` interface for unit tests
- Use `t.TempDir()` for file operations
- Test fixtures: sample gcloud outputs, existing config files
- Test cases:
  - gcloud CLI not installed (error)
  - Account discovery with single account
  - Account discovery with multiple accounts
  - Configuration listing
  - Configuration creation
  - Configuration activation
  - ADC file generation
  - Backup creation
  - Dry run mode

## Error Handling

| Scenario                     | Behavior                                                             |
| ---------------------------- | -------------------------------------------------------------------- |
| No gcloud installed          | Error: "gcloud CLI not found. Install Google Cloud SDK first."       |
| No authenticated accounts    | Error: "No authenticated GCP accounts. Run 'gcloud auth login'."     |
| Configuration already exists | Error: "Configuration 'X' already exists. Use --force to overwrite." |
| Delete active configuration  | Error: "Cannot delete active configuration. Activate another first." |
| ADC generation fails         | Error: "Failed to generate ADC: <details>"                           |

## Decisions

1. **Shell Out to gcloud**: Use gcloud CLI for all operations to ensure compatibility and simplicity.
2. **Configuration-Based**: Store account/project mappings in config, not discovered dynamically.
3. **No Auto-Login**: Prompt user to login manually; keeps provider stateless.
4. **Backup Always**: Always backup existing ADC file unless `--force` is set.
5. **ADC from gcloud**: Extract ADC from gcloud's cache rather than regenerating.
6. **User/Service Detection**: Show account type in discovery output.

## Implementation Phases

### Phase 1: Core Discovery

- [ ] Gcloud client interface and mock
- [ ] Account discovery
- [ ] Configuration listing
- [ ] Basic output formatting

### Phase 2: Configuration Management

- [ ] Create configuration
- [ ] Activate configuration
- [ ] Delete configuration
- [ ] CLI integration

### Phase 3: ADC Generation

- [ ] ADC file reading from gcloud cache
- [ ] Backup mechanism
- [ ] ADC file writing
- [ ] Demo mode
- [ ] Documentation
