# Granted Provider Specification

## Overview

The Granted provider generates the `~/.granted/config` file with a simple set of defaults. It does not handle AWS profile or registry generation; the scope is limited to creating the Granted config file using the new provider architecture.

## Features

### Granted Config Generation

1. Writes a default Granted configuration file at the configured path.
2. Ensures the parent directory exists.
3. Respects `--dry-run` and `--force` semantics shared across providers.

## Configuration

```yaml
providers:
  granted:
    enabled: true
    config_path: ~/.granted/config
    credential_process_auto_login: true
```

## CLI Interface

```bash
# Generate Granted config
cfgctl generate granted

# Dry run
cfgctl generate granted --dry-run

# Overwrite existing config
cfgctl generate granted --force
```

## Implementation

### Directory Structure

```
internal/providers/granted/
  config.go         # Config struct, validation, defaults
  provider.go       # Provider implementation
  provider_test.go  # tests
  testdata/         # fixtures (if needed)
```

### Config Type

```go
type Config struct {
    Enabled                     bool   `yaml:"enabled"`
    ConfigPath                  string `yaml:"config_path"`
    CredentialProcessAutoLogin  bool   `yaml:"credential_process_auto_login"`
}
```

### Default Output

The generated config file should include the following defaults:

```
DefaultBrowser = "STDOUT"
CustomBrowserPath = ""
CustomSSOBrowserPath = ""
Ordering = ""
ExportCredentialSuffix = ""
DisableUsageTips = true
CredentialProcessAutoLogin = true
```

### Provider Behavior

- Validate that `config_path` is absolute and non-empty.
- When the target file exists and `--force` is not set, skip generation and return a warning.
- When `--dry-run` is set, do not write files; return planned actions in metadata.
- Ensure the output directory exists before writing.
- Register the provider in `internal/cli/root.go` using default config when missing.

## Dependencies

- Standard library only.

Required tools:

- `granted`

## Testing

- Verify file creation at default and custom paths.
- Verify content matches defaults, including `CredentialProcessAutoLogin`.
- Verify `--force` overwrites existing config.
- Verify `--dry-run` does not write files.
- Use `t.TempDir()` for file operations.

## Decisions

1. **Scope**: Only Granted config file generation, no AWS profile or registry features.
2. **Defaults**: Enable `CredentialProcessAutoLogin` by default for smoother CLI usage.
3. **Templating**: Use simple string assembly rather than OS-specific templates.
