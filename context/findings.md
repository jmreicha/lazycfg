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

## User Preferences

- Use `kubectl` and `k9s` as required tools for the Kubernetes provider and skip with a warning when missing.

## Patterns That Work

- Use core-level tool discovery with common PATH fallbacks and skip providers with missing tools.

## Patterns That Don't Work

- Assume `.claude/skills` locations without verifying via repo listing.

## Domain Notes

- Tool discovery for providers is now centralized in `internal/core/engine.go` via `providerMissingTools`.
