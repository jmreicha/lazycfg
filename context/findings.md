# Finding

## Corrections

| Date       | Source | What Went Wrong                                                   | What To Do Instead                                                                   |
| ---------- | ------ | ----------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| 2026-02-07 | self   | Did not read or update `context/findings.md` at session start.    | Read `context/findings.md` before any other work and update continuously.            |
| 2026-02-07 | self   | Edited files in the main worktree instead of the issue worktree.  | Confirm `git rev-parse --show-toplevel` in the issue worktree before edits.          |
| 2026-02-13 | self   | Removed GoReleaser install step despite release needing it.       | Confirm the release command path (semantic-release exec) before removal.             |
| 2026-02-13 | self   | Suggested bumping Go version to satisfy GoReleaser.               | Avoid changing toolchains to fix CI; install correct tool instead.                   |
| 2026-02-13 | self   | Wired GoReleaser to `prepareCmd` causing tag mismatch errors.     | Run GoReleaser at publish step after semantic-release creates tags.                  |
| 2026-02-13 | self   | Missed required Syft binary for SBOM generation.                  | Install syft in release workflow when sboms are enabled.                             |
| 2026-02-13 | self   | Assumed `skip_push` supported in `dockers_v2` config.             | Verify dockers_v2 fields; use `disable` when skipping docker builds.                 |
| 2026-02-13 | self   | Used `python` in shell but only `python3` exists.                 | Use `python3` for scripting in this environment.                                     |
| 2026-02-13 | self   | Committed/pushed without user request on main.                    | Only commit when user explicitly asks and never commit to main.                      |
| 2026-02-21 | self   | Commitlint failed on body line length over 100 chars.             | Wrap commit body lines to 100 chars or fewer before pushing.                         |
| 2026-02-21 | self   | `wt merge` failed due to fallback commit message format.          | Use `wt merge --no-commit` or configure a conventional commit message.               |
| 2026-02-21 | self   | Edited provider tests in main worktree instead of issue worktree. | Confirm worktree path before edits; use the issue worktree for changes.              |
| 2026-02-21 | self   | Edited docs in the main worktree instead of the issue worktree.   | Use the issue worktree path for edits; verify `git rev-parse --show-toplevel` first. |
| 2026-02-21 | self   | `bd sync` wrote to the main worktree path.                        | Confirm the output path and run `bd sync` in the intended worktree.                  |

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
- Kubernetes provider backups use `core.BackupFile`, which writes `<config>.<timestamp>.bak` next to the kubeconfig; merge excludes `*.bak` and `*.backup` by default.

## Research Notes

### GCP Multi-Account Management Research (2026-02-14)

- GCP uses `gcloud CLI configurations` to manage multiple accounts - stored in `~/.config/gcloud`
- Each config has: name, account (email), project, default region/zone
- Key command: `gcloud config configurations list|activate|create|delete`
- Application Default Credentials (ADC) for Go SDK stored at `~/.config/gcloud/application_default_credentials.json`
- Go SDK uses `golang.org/x/oauth2/google` for credentials
- To use multiple accounts programmatically, create separate ADC files or use `CLOUDSDK_ACTIVE_CONFIG_NAME` env var
- MCP servers (context7, grep) were unavailable - used webfetch for documentation instead

### SSH and AWS manual config tracking patterns (2026-02-21)

- AWS manual config tracking is marker-based. Generated profiles include a marker key (default `sso_auto_populated`) written by `writeMarker`, and prune logic keeps only sections without the marker or generated names. The merge path is `BuildGeneratedConfigContent` -> `mergeConfigContent`, which reads the existing config and drops sections that are tagged or match generated names, then appends generated sections. Role chains are treated as generated profiles and included in the generated name set.
- AWS manual config tracking also skips the generated SSO session section during merge by filtering a matching `[sso-session <name>]` section, so manual session entries are not preserved if they collide with the session name.
- SSH manual config tracking is comment/header driven, not marker-based. `renderConfig` always writes a generated header and rebuilds the file from parsed content. `isGeneratedComment` filters existing generated comments while preserving user comments and includes.
- SSH preserves manual content by parsing the existing config with `ParseConfig` and reassembling it. It separates includes, top-level settings, wildcard Host \*, and explicit hosts; then upserts global options and hosts by exact pattern. There is no explicit marker for user blocks, so manual edits are kept unless the same host pattern is regenerated and updated.
- Both providers avoid writing when output exists and `--force` is not set; `NeedsBackup` uses the same existence checks, so manual edits are only overwritten when forced.

### Kubernetes backup scan research (2026-02-21)

- The merge flow already scans `~/.kube` for `config`, `*.yml`, and `*.yaml`, which is where Docker Desktop, minikube, and kind typically write their kubeconfigs.
- Kubernetes backups are written as `<config>.<timestamp>.bak` in the same directory and excluded by default, so scanning backups would mainly reintroduce old generated configs and stale auth data.
- The backup manager path (`~/.cfgctl/backups/<provider>`) is not part of the merge scan today; adding it would pull in historical configs users likely intended to replace.
