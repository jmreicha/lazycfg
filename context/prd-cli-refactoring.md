# PRD: CLI Framework Refactoring (Kong + Lipgloss)

## Overview

Refactor cfgctl's CLI from Cobra to Kong for cleaner command structure with better flag organization, and add Lipgloss for polished terminal output. This addresses confusing flag naming (e.g., `--kube-roles` which is EKS-specific) and consolidates provider-specific flags to their respective generate commands.

## Goals

- Migrate from Cobra to Kong for declarative, type-safe CLI parsing
- Add Lipgloss for styled, professional terminal output
- Rename EKS-specific flags to be explicit (e.g., `--eks-roles`, `--eks-regions`)
- Move provider-specific flags to their respective generate commands
- Maintain flat command structure (`cfgctl generate aws`, `cfgctl generate kubernetes`)
- Preserve all existing global flags

## Quality Gates

These commands must pass for every user story:

- `go test ./...` - All tests pass
- `golangci-lint run` - Linting passes
- `go build ./...` - Build succeeds

## User Stories

### US-001: Add Kong and Lipgloss dependencies

**Description:** As a developer, I want the required dependencies added so I can begin the migration.

**Acceptance Criteria:**

- [ ] Add `github.com/alecthomas/kong` to go.mod
- [ ] Add `github.com/charmbracelet/lipgloss` to go.mod
- [ ] Run `go mod tidy` to resolve dependencies

### US-002: Create Kong CLI structure with global flags

**Description:** As a user, I want all existing global flags available across all commands.

**Acceptance Criteria:**

- [ ] Create new `internal/cli/kong.go` with Kong CLI struct
- [ ] `--config` flag for config file path
- [ ] `--dry-run` flag to simulate without changes
- [ ] `--debug` flag for debug logging
- [ ] `-v, --verbose` flag for verbose provider output
- [ ] `--no-backup` flag to skip backup creation
- [ ] All global flags apply to all commands
- [ ] Environment variable support via Kong env tags (CFGCTL_CONFIG, CFGCTL_DRY_RUN, CFGCTL_DEBUG, etc.)

### US-003: Migrate generate command with per-provider flags

**Description:** As a user, I want provider-specific flags grouped under their generate subcommands, not at the top level.

**Acceptance Criteria:**

- [ ] `cfgctl generate aws` with AWS-specific flags (credential-process, credentials, demo, prefix, prune, roles, sso-url, sso-region, template)
- [ ] `cfgctl generate kubernetes` with Kube-specific flags (merge, merge-only, eks-regions, eks-roles)
- [ ] `cfgctl generate ssh` with SSH-specific flag (config-path)
- [ ] Positional args for provider names work: `cfgctl generate aws kubernetes`
- [ ] `cfgctl generate all` still works
- [ ] `--force` flag available on generate command
- [ ] Invalid provider name errors with list of valid providers
- [ ] Conflicting flag combinations validated (--merge-only + --eks-regions should error)

### US-004: Rename EKS-specific flags

**Description:** As a user, I want EKS-specific flags clearly named so I understand they only apply to EKS.

**Acceptance Criteria:**

- [ ] `--kube-regions` renamed to `--eks-regions`
- [ ] `--kube-roles` renamed to `--eks-roles`
- [ ] Flags only visible/valid for `cfgctl generate kubernetes`
- [ ] Help text clarifies these are for EKS discovery

### US-005: Migrate list command

**Description:** As a user, I want to list providers with `cfgctl list`.

**Acceptance Criteria:**

- [ ] `cfgctl list` shows registered providers
- [ ] Output uses Lipgloss styling

### US-006: Migrate clean command

**Description:** As a user, I want to clean provider configs with `cfgctl clean <provider>`.

**Acceptance Criteria:**

- [ ] `cfgctl clean aws` removes AWS-generated configs
- [ ] `cfgctl clean kubernetes` removes kube-generated configs
- [ ] Multiple providers: `cfgctl clean aws kubernetes`
- [ ] Output uses Lipgloss styling

### US-007: Migrate validate command

**Description:** As a user, I want to validate provider prerequisites with `cfgctl validate`.

**Acceptance Criteria:**

- [ ] `cfgctl validate` checks all provider prerequisites
- [ ] Output uses Lipgloss styling

### US-008: Add Lipgloss output styling

**Description:** As a user, I want polished terminal output with colors and formatting.

**Acceptance Criteria:**

- [ ] Create `internal/cli/styles.go` with Lipgloss style definitions
- [ ] Provider names styled (bold/color)
- [ ] Success/error/warning messages styled distinctly
- [ ] File paths styled
- [ ] Respects NO_COLOR env var
- [ ] Respects non-TTY output (disables colors)

### US-009: Wire Kong CLI to main entrypoint

**Description:** As a developer, I want the new Kong CLI to be the active CLI.

**Acceptance Criteria:**

- [ ] Update `cmd/cfgctl/main.go` to use Kong parser
- [ ] Remove Cobra dependency (delete old CLI files after migration)
- [ ] All existing functionality preserved

### US-010: Update tests for Kong CLI

**Description:** As a developer, I want tests updated for the new CLI structure.

**Acceptance Criteria:**

- [ ] `internal/cli/commands_test.go` updated for Kong
- [ ] `internal/cli/generate_test.go` updated for Kong
- [ ] Flag tests updated for new flag names
- [ ] All tests pass

## Functional Requirements

- FR-1: CLI must support all existing commands (generate, list, clean, validate, version)
- FR-2: Global flags: --config, --dry-run, --debug, --verbose, --no-backup
- FR-3: Provider-specific flags must only appear on relevant commands
- FR-4: Help output must be clear and well-formatted
- FR-5: Error messages must be styled and actionable
- FR-6: Must respect terminal capabilities (color, width)
- FR-7: Invalid provider name must error with list of valid providers
- FR-8: Validate conflicting flag combinations (e.g., --merge-only with --eks-regions)
- FR-9: Support environment variables via Kong env tags (e.g., CFGCTL_CONFIG, CFGCTL_DRY_RUN)
- FR-10: Version output remains plain text (no Lipgloss styling)

## Non-Goals

- Interactive TUI mode (Bubble Tea integration)
- Shell completion generation (can be added later)
- Config file migration (format unchanged)
- Breaking existing config file compatibility

## Technical Considerations

- Use Kong's `Run() error` method pattern for command handling
- Use Kong's `BeforeApply` hooks for flag validation
- Use Lipgloss's `AdaptiveColor` for light/dark terminal support
- Consider Kong's help customization for styled help output
- Keep existing `internal/core` components unchanged

## Success Metrics

- All tests pass
- Lint passes
- `cfgctl --help` shows clean, organized help with global flags
- `cfgctl generate --help` shows global flags + generate-specific flags
- `cfgctl generate aws --help` shows only AWS flags + global flags
- `cfgctl generate kubernetes --help` shows only Kube flags with EKS-specific names + global flags
- Output is visually polished with Lipgloss

## Open Questions

- Should Kong help output be styled with Lipgloss? (defer until manual testing)
