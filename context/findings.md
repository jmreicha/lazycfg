# Finding

## Corrections

| Date       | Source | What Went Wrong                                                  | What To Do Instead                                                          |
| ---------- | ------ | ---------------------------------------------------------------- | --------------------------------------------------------------------------- |
| 2026-02-07 | self   | Did not read or update `context/findings.md` at session start.   | Read `context/findings.md` before any other work and update continuously.   |
| 2026-02-07 | self   | Edited files in the main worktree instead of the issue worktree. | Confirm `git rev-parse --show-toplevel` in the issue worktree before edits. |
| 2026-02-13 | self   | Removed GoReleaser install step despite release needing it.      | Confirm the release command path (semantic-release exec) before removal.    |
| 2026-02-13 | self   | Suggested bumping Go version to satisfy GoReleaser.              | Avoid changing toolchains to fix CI; install correct tool instead.          |
| 2026-02-13 | self   | Wired GoReleaser to `prepareCmd` causing tag mismatch errors.    | Run GoReleaser at publish step after semantic-release creates tags.         |
| 2026-02-13 | self   | Missed required Syft binary for SBOM generation.                 | Install syft in release workflow when sboms are enabled.                    |
| 2026-02-13 | self   | Assumed `skip_push` supported in `dockers_v2` config.            | Verify dockers_v2 fields; use `disable` when skipping docker builds.        |
| 2026-02-13 | self   | Used `python` in shell but only `python3` exists.                | Use `python3` for scripting in this environment.                            |
| 2026-02-13 | self   | Committed/pushed without user request on main.                   | Only commit when user explicitly asks and never commit to main.             |
| 2026-02-21 | self   | Commitlint failed on body line length over 100 chars.            | Wrap commit body lines to 100 chars or fewer before pushing.                |

## User Preferences

- Use `kubectl` and `k9s` as required tools for the Kubernetes provider and skip with a warning when missing.

## Patterns That Work

- Use core-level tool discovery with common PATH fallbacks and skip providers with missing tools.
- Gate backups with a provider-level check so we only create backups when generation will write.
- For full renames, update module path, CLI binary, config paths, release metadata, and docs together.

## Patterns That Don't Work

- Assume `.claude/skills` locations without verifying via repo listing.

## Domain Notes

- Tool discovery for providers is now centralized in `internal/core/engine.go` via `providerMissingTools`.
- SSH provider uses comment-based markers to distinguish generated vs user content; a generic `BlockPreserver` interface is planned for `internal/core/` to make this reusable across providers (steampipe first, then SSH migration).
- Provider interface includes `BackupDecider` (`NeedsBackup`) for conditional backups; engine handles backup/rollback orchestration.

## Research Notes

### GCP Multi-Account Management Research (2026-02-14)

- GCP uses `gcloud CLI configurations` to manage multiple accounts - stored in `~/.config/gcloud`
- Each config has: name, account (email), project, default region/zone
- Key command: `gcloud config configurations list|activate|create|delete`
- Application Default Credentials (ADC) for Go SDK stored at `~/.config/gcloud/application_default_credentials.json`
- Go SDK uses `golang.org/x/oauth2/google` for credentials
- To use multiple accounts programmatically, create separate ADC files or use `CLOUDSDK_ACTIVE_CONFIG_NAME` env var
- MCP servers (context7, grep) were unavailable - used webfetch for documentation instead
