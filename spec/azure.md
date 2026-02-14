# Azure Provider Specification

## Overview

The Azure provider generates the Azure CLI configuration file at `$AZURE_CONFIG_DIR/config` (default `$HOME/.azure/config`). It manages default subscription and tenant settings, enabling seamless switching between multiple Azure subscriptions and tenants.

## Features

### Azure CLI Config Generation

1. Writes a default Azure CLI configuration file at the configured path.
2. Sets default subscription and tenant for commands.
3. Configures output format and other sensible defaults.
4. Respects `--dry-run` and `--force` semantics shared across providers.

## Configuration

```yaml
providers:
  azure:
    enabled: true
    config_path: ~/.azure/config
    defaults:
      subscription: ""
      tenant: ""
      location: ""
      group: ""
    output: json
    disable_confirm_prompt: true
```

## CLI Interface

```bash
# Generate Azure config
cfgctl generate azure

# Dry run
cfgctl generate azure --dry-run

# Overwrite existing config
cfgctl generate azure --force
```

## Implementation

### Directory Structure

```
internal/providers/azure/
  config.go         # Config struct, validation, defaults
  provider.go       # Provider implementation
  provider_test.go  # tests
  testdata/         # fixtures (if needed)
```

### Config Type

```go
type Config struct {
    Enabled              bool   `yaml:"enabled"`
    ConfigPath           string `yaml:"config_path"`
    Defaults             DefaultsConfig `yaml:"defaults"`
    Output               string `yaml:"output"`
    DisableConfirmPrompt bool   `yaml:"disable_confirm_prompt"`
}

type DefaultsConfig struct {
    Subscription string `yaml:"subscription"`
    Tenant      string `yaml:"tenant"`
    Location    string `yaml:"location"`
    Group       string `yaml:"group"`
}
```

### Default Output

The generated config file should include the following defaults:

```ini
[core]
output = json
disable_confirm_prompt = true

[defaults]
group =
location =
```

### Provider Behavior

- Validate that `config_path` is absolute and non-empty.
- When the target file exists and `--force` is not set, skip generation and return a warning.
- When `--dry-run` is set, do not write files; return planned actions in metadata.
- Ensure the output directory exists before writing.
- Register the provider in `internal/cli/root.go` using default config when missing.
- Support subscription and tenant selection via `defaults.subscription` and `defaults.tenant`.

## Dependencies

- Standard library only.

Required tools:

- `az` (Azure CLI)

## Testing

- Verify file creation at default and custom paths.
- Verify content matches defaults.
- Verify `--force` overwrites existing config.
- Verify `--dry-run` does not write files.
- Use `t.TempDir()` for file operations.

## Decisions

1. **Scope**: Only Azure CLI config file generation, no credential management.
2. **Defaults**: Set `disable_confirm_prompt` and `output=json` for scripting.
3. **Templating**: Use simple string assembly rather than templates.
4. **Subscription/Tenant**: Use `az account set` at runtime rather than baking into config (more flexible).
