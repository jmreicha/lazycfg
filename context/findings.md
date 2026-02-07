# Finding

## Corrections

| Date       | Source | What Went Wrong                                                | What To Do Instead                                                        |
| ---------- | ------ | -------------------------------------------------------------- | ------------------------------------------------------------------------- |
| 2026-02-07 | self   | Did not read or update `context/findings.md` at session start. | Read `context/findings.md` before any other work and update continuously. |

## User Preferences

- Use `kubectl` and `k9s` as required tools for the Kubernetes provider and skip with a warning when missing.

## Patterns That Work

- Use core-level tool discovery with common PATH fallbacks and skip providers with missing tools.

## Patterns That Don't Work

- Assume `.claude/skills` locations without verifying via repo listing.

## Domain Notes

- Tool discovery for providers is now centralized in `internal/core/engine.go` via `providerMissingTools`.
