# PRD: CLI Refactoring with Cobra + Viper

## Overview

Refactor cfgctl's CLI to use Cobra + Viper (matching Docker, Kubernetes, GitHub CLI patterns) for cleaner command structure, better flag organization, and environment variable support. This addresses confusing flag naming (e.g., `--kube-roles` which is EKS-specific) and adds proper configuration via environment variables.

## Goals

- Use Cobra + Viper (matching Docker, Kubernetes, GitHub CLI patterns)
- Add Lipgloss for styled terminal output
- Rename EKS-specific flags to be explicit (e.g., `--eks-roles`, `--eks-regions`)
- Move provider-specific flags to their respective generate commands
- Maintain flat command structure (`cfgctl generate aws`, `cfgctl generate kubernetes`)
- Add environment variable support (CFGCTL\_\*)
- Preserve all existing global flags

## Quality Gates

These commands must pass for every user story:

- `go test ./...` - All tests pass
- `golangci-lint run` - Linting passes
- `go build ./...` - Build succeeds

## User Stories

### US-001: Add Viper dependency

**Description:** As a developer, I want Viper added so I can handle configuration from env vars and config files.

**Acceptance Criteria:**

- [ ] Add github.com/spf13/viper to go.mod
- [ ] Run go mod tidy to resolve dependencies
- [ ] go build ./... succeeds

### US-002: Refactor root command with global flags and Viper

**Description:** As a user, I want all existing global flags available across all commands with env var support.

**Acceptance Criteria:**

- [ ] Global flags: --config, --dry-run, --debug, -v/--verbose, --no-backup
- [ ] Environment variable support: CFGCTL_CONFIG, CFGCTL_DRY_RUN, CFGCTL_DEBUG, CFGCTL_VERBOSE, CFGCTL_NO_BACKUP
- [ ] Viper reads from config file, env vars, and flags (in precedence order)
- [ ] Global flags apply to all commands
- [ ] go test ./... passes

### US-003: Add generate command with provider flags

**Description:** As a user, I want provider-specific flags grouped on the generate command.

**Acceptance Criteria:**

- [ ] cfgctl generate aws with AWS-specific flags (credential-process, credentials, demo, prefix, prune, roles, sso-url, sso-region, template)
- [ ] cfgctl generate kubernetes with Kube-specific flags (merge, merge-only, eks-regions, eks-roles)
- [ ] cfgctl generate ssh with SSH-specific flag (config-path)
- [ ] Positional args work: cfgctl generate aws kubernetes
- [ ] cfgctl generate all still works
- [ ] --force flag on generate command
- [ ] go test ./... passes

### US-004: Rename EKS-specific flags

**Description:** As a user, I want EKS-specific flags clearly named so I understand they only apply to EKS.

**Acceptance Criteria:**

- [ ] --kube-regions renamed to --eks-regions
- [ ] --kube-roles renamed to --eks-roles
- [ ] Help text clarifies these are for EKS discovery
- [ ] go test ./... passes

### US-005: Add provider validation with helpful errors

**Description:** As a user, I want clear error messages when I specify an invalid provider.

**Acceptance Criteria:**

- [ ] Invalid provider name errors with list of valid providers
- [ ] Conflicting flag combinations validated (--merge-only + --eks-regions should error)
- [ ] go test ./... passes

### US-006: Add Lipgloss output styling

**Description:** As a user, I want polished terminal output with colors and formatting.

**Acceptance Criteria:**

- [ ] Add github.com/charmbracelet/lipgloss to go.mod
- [ ] Create internal/cli/styles.go with Lipgloss style definitions
- [ ] Provider names styled (bold/color)
- [ ] Success/error/warning messages styled distinctly
- [ ] File paths styled
- [ ] Respects NO_COLOR env var and non-TTY output
- [ ] go test ./... passes

### US-007: Migrate list command

**Description:** As a user, I want to list providers with cfgctl list.

**Acceptance Criteria:**

- [ ] cfgctl list shows registered providers
- [ ] Output uses Lipgloss styling
- [ ] go test ./... passes

### US-008: Migrate clean command

**Description:** As a user, I want to clean provider configs with cfgctl clean provider.

**Acceptance Criteria:**

- [ ] cfgctl clean aws removes AWS-generated configs
- [ ] cfgctl clean kubernetes removes kube-generated configs
- [ ] Multiple providers: cfgctl clean aws kubernetes
- [ ] Output uses Lipgloss styling
- [ ] go test ./... passes

### US-009: Migrate validate command

**Description:** As a user, I want to validate provider prerequisites with cfgctl validate.

**Acceptance Criteria:**

- [ ] cfgctl validate checks all provider prerequisites
- [ ] Output uses Lipgloss styling
- [ ] go test ./... passes

### US-010: Wire new CLI structure to main

**Description:** As a developer, I want the refactored CLI to be wired to main.go.

**Acceptance Criteria:**

- [ ] All commands (generate, list, clean, validate, version) wired to root
- [ ] Viper properly initialized at startup
- [ ] All existing functionality preserved
- [ ] go test ./... passes

### US-011: Update tests

**Description:** As a developer, I want tests updated for the refactored CLI.

**Acceptance Criteria:**

- [ ] Test flag names updated for new EKS-specific flags
- [ ] Test provider validation
- [ ] All tests pass

## Functional Requirements

- FR-1: CLI must support all existing commands (generate, list, clean, validate, version)
- FR-2: Global flags: --config, --dry-run, --debug, --verbose, --no-backup
- FR-3: Environment variables: CFGCTL_CONFIG, CFGCTL_DRY_RUN, CFGCTL_DEBUG, CFGCTL_VERBOSE, CFGCTL_NO_BACKUP
- FR-4: Provider-specific flags only on relevant commands
- FR-5: Invalid provider shows list of valid providers
- FR-6: Flag conflict validation

## Non-Goals

- Interactive TUI mode
- Shell completion generation
- Config file migration

## Technical Considerations

- Use Viper for all config: flags + env vars + config files
- Set env prefix: viper.SetEnvPrefix("CFGCTL")
- Config precedence: flag > env > config file > default
- Use PersistentPreRunE for Viper initialization
- Keep existing internal/core components unchanged
