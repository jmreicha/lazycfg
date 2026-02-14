# Steampipe Provider Specification

## Overview

The Steampipe provider generates `~/.steampipe/config/aws.spc` by reading existing AWS profiles from `~/.aws/config` and creating corresponding Steampipe connection blocks in HCL. It depends on the AWS provider having already generated a valid AWS config.

## Goals

- Generate Steampipe AWS connection config from existing AWS profiles
- Create an aggregator connection that queries all generated connections
- Backup existing config before overwriting
- Fail gracefully when AWS config is missing
- Fail when `steampipe` CLI is not on PATH

## Non-Goals

- Managing Steampipe plugins or plugin installation
- Generating connections for non-AWS plugins (GCP, Azure, GitHub planned for later)
- SSO login or credential management
- Steampipe query execution

## Features

### AWS Profile Discovery

1. Parse `~/.aws/config` to extract profile names
2. If the AWS config file does not exist, log a warning and skip generation (no error)
3. For each profile, generate a Steampipe connection block
4. Optionally filter profiles by name or prefix

### Connection Generation

Each AWS profile becomes a Steampipe connection:

```hcl
connection "aws_prod_account_admin_access" {
  plugin  = "aws"
  profile = "prod-account/AdminAccess"
  regions = ["*"]
}
```

Profile names are sanitized for HCL identifiers (replace non-alphanumeric characters with underscores).

Per-profile region overrides can be specified in config. When a profile matches an entry in `profile_regions`, that region list is used instead of the default.

### Aggregator Connection

An aggregator connection is generated to query all connections:

```hcl
connection "aws_all" {
  plugin      = "aws"
  type        = "aggregator"
  connections = ["aws_*"]
}
```

### Custom Block Preservation

Users may add custom connection blocks or modify generated ones. The provider preserves user-managed blocks across regeneration using a generic marker-based system designed for reuse by other providers.

#### Marker Convention

Generated blocks are tagged with a comment marker immediately above the block:

```hcl
# managed-by: cfgctl
connection "aws_prod_account_admin_access" {
  plugin  = "aws"
  profile = "prod-account/AdminAccess"
  regions = ["*"]
}
```

Blocks without the `# managed-by: cfgctl` marker are considered user-managed and are preserved as-is during regeneration.

#### Merge Flow

1. Parse existing config file into blocks (each block = optional preceding comments + connection block)
2. Separate blocks into managed (has marker) and user-managed (no marker)
3. Generate new managed blocks from AWS profiles
4. Merge: user-managed blocks are preserved in their original position, managed blocks are replaced with the new set
5. Write combined output

#### Generic Block Preservation Interface

To enable reuse across providers, block preservation is implemented as a shared utility in `internal/core/`:

```go
type ManagedBlock struct {
    Comment string // preceding comment lines
    Content string // the block content
    Managed bool   // true if tagged with marker
    Name    string // block identifier (e.g., connection name)
}

type BlockPreserver interface {
    Parse(content string) []ManagedBlock
    Merge(existing []ManagedBlock, generated []ManagedBlock) []ManagedBlock
    Render(blocks []ManagedBlock) string
}
```

The marker string is configurable per provider. The SSH provider's existing comment-based preservation can be migrated to this interface in a future refactor.

### Tool Requirement

The `steampipe` CLI must be available on PATH. If not found, the provider skips with a warning, consistent with other providers.

### AWS Config Dependency

If `~/.aws/config` does not exist or is empty, the provider returns a warning and skips generation. It does not return an error; other providers can still run.

### Backup Strategy

Before writing to `~/.steampipe/config/aws.spc`:

1. Check if file exists
2. If exists, create backup: `~/.steampipe/config/aws.spc.bak.YYYYMMDD-HHMMSS`
3. Write new config

Uses the same `NeedsBackup()` / `Backup()` pattern as other providers.

## Configuration

```yaml
providers:
  steampipe:
    enabled: true
    aws_config_path: ~/.aws/config # source of AWS profiles
    config_path: ~/.steampipe/config/aws.spc # output file
    connection_prefix: aws_ # prefix for connection names
    aggregator_name: aws_all # name of the aggregator connection
    regions: ["*"] # default regions for each connection
    profiles: [] # empty = all profiles
    backup_enabled: true
    backup_dir: ~/.steampipe/config # where to store .bak files

    # Per-profile region overrides (optional)
    profile_regions:
      prod-account/AdminAccess: ["us-east-1", "us-west-2"]
      staging-account/ReadOnly: ["us-east-1"]
```

## CLI Interface

```bash
# Generate Steampipe config from AWS profiles
cfgctl generate steampipe

# Filter to specific profiles
cfgctl generate steampipe --profiles prod,staging

# Dry run
cfgctl generate steampipe --dry-run

# Force overwrite (no backup prompt)
cfgctl generate steampipe --force
```

## Implementation

### Directory Structure

```
internal/providers/steampipe/
  config.go         # Config struct, validation, defaults
  provider.go       # Provider implementation
  provider_test.go  # Tests
  testdata/         # Fixtures
```

### Core Types

```go
type Config struct {
    AggregatorName   string            `yaml:"aggregator_name"`
    AWSConfigPath    string            `yaml:"aws_config_path"`
    BackupDir        string            `yaml:"backup_dir"`
    BackupEnabled    bool              `yaml:"backup_enabled"`
    ConfigPath       string            `yaml:"config_path"`
    ConnectionPrefix string            `yaml:"connection_prefix"`
    Enabled          bool              `yaml:"enabled"`
    ProfileRegions   map[string][]string `yaml:"profile_regions"`
    Profiles         []string          `yaml:"profiles"`
    Regions          []string          `yaml:"regions"`
}
```

### Generation Flow

1. Validate config (paths non-empty, etc.)
2. Check AWS config file exists; if not, warn and return
3. Parse AWS config to extract profile names
4. Filter profiles if configured
5. If output file exists, parse into managed/user-managed blocks
6. For each profile:
   - Sanitize name to valid HCL identifier
   - Resolve regions (per-profile override or default)
   - Generate connection block with `# managed-by: cfgctl` marker
7. Generate aggregator connection block (also marked)
8. Merge: replace managed blocks, preserve user-managed blocks
9. Write combined output to config path

### Profile Name Sanitization

AWS profile names may contain characters invalid in HCL identifiers. Sanitize by:

1. Replacing `/`, `-`, `.`, and spaces with `_`
2. Prepending `connection_prefix` (default: `aws_`)
3. Lowercasing the result

Example: `prod-account/AdminAccess` becomes `aws_prod_account_adminaccess`

## Dependencies

- Standard library only (no HCL library needed; output is simple enough for string assembly)

Required tools:

- `steampipe`

## Testing

- Use `t.TempDir()` for file operations
- Test fixtures: sample AWS config files, sample `.spc` files with mixed managed/user blocks
- Test cases:
  - Generation with valid AWS config
  - Missing AWS config (graceful skip)
  - Empty AWS config (graceful skip)
  - Profile filtering
  - Profile name sanitization
  - Per-profile region overrides
  - Aggregator connection generation
  - Custom block preservation (user blocks survive regeneration)
  - Custom block ordering (user blocks stay in original position)
  - Backup creation
  - Dry run mode
  - Force overwrite
  - Missing steampipe CLI (skip with warning)
  - Generic BlockPreserver unit tests (in `internal/core/`)

## Error Handling

| Scenario                     | Behavior                                                                                       |
| ---------------------------- | ---------------------------------------------------------------------------------------------- |
| `steampipe` not on PATH      | Skip provider with warning                                                                     |
| AWS config missing           | Skip provider with warning: "AWS config not found at <path>, skipping steampipe generation"    |
| AWS config empty/no profiles | Skip with warning: "No AWS profiles found"                                                     |
| Invalid config_path          | Error: "config_path is required"                                                               |
| Backup failure               | Error: "failed to backup steampipe config: <error>"                                            |
| Malformed existing .spc      | Warning: "failed to parse existing config, user blocks may not be preserved"; regenerate fully |

## Decisions

1. **String assembly over HCL library**: Output format is simple; avoids adding a dependency.
2. **AWS config as source**: Reads profiles from the AWS config file rather than re-discovering from SSO.
3. **Graceful degradation**: Missing AWS config is a warning, not an error, so other providers still run.
4. **Wildcard regions by default**: Default `["*"]` lets Steampipe query all regions; per-profile overrides available.
5. **Aggregator by default**: Always generate an aggregator connection for convenience.
6. **Tool check via engine**: Uses the existing `providerMissingTools` pattern in `engine.go`.
7. **Comment-based markers**: `# managed-by: cfgctl` identifies generated blocks; unmarked blocks are user-managed.
8. **Generic block preservation**: `BlockPreserver` in `internal/core/` is reusable across providers; SSH provider can migrate later.
9. **Future plugins**: GCP, Azure, GitHub connection generation planned for later phases.

## Implementation Phases

### Phase 1: Generic Block Preservation

- [ ] `BlockPreserver` interface and HCL-style implementation in `internal/core/`
- [ ] Marker-based parse/merge/render logic
- [ ] Unit tests for block preservation

### Phase 2: Core Generation

- [ ] Config struct with validation and defaults
- [ ] AWS config parsing (extract profile names)
- [ ] Connection block generation with markers
- [ ] Per-profile region overrides
- [ ] Aggregator connection generation
- [ ] Profile name sanitization
- [ ] Unit tests

### Phase 3: Integration

- [ ] Provider registration in CLI
- [ ] Tool check (`steampipe`) in engine
- [ ] Backup support via `NeedsBackup()` / `Backup()`
- [ ] Custom block preservation during generation
- [ ] Dry run and force flag support
- [ ] CLI flag integration (--profiles)

### Phase 4: Polish

- [ ] Profile filtering
- [ ] Documentation

### Future

- [ ] GCP plugin connection generation
- [ ] Azure plugin connection generation
- [ ] GitHub plugin connection generation
- [ ] Migrate SSH provider to generic `BlockPreserver`
