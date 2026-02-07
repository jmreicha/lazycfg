# AI Agent Configuration

This repository is configured to work with multiple AI coding agents. Each agent has its own configuration file with shared coding standards and project-specific context.

## Project Overview

This is a Golang project with automated tooling for code quality, conventional commits, and semantic versioning. The project is configured for modern development practices with full CI/CD automation.

## Agent Effectiveness Guidelines

Based on [Anthropic's guide to long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents):

- **Follow recommendations precisely** - Read entire sources before proposing solutions; don't paraphrase without justification
- **If corrected, acknowledge and fix** - Don't defend substitutions that contradict the source
- **Work on one task at a time** - Avoid scope creep and doing too much at once
- **Documentation is part of implementation** - When adding functionality, update all related docs (module docs, function docs, user-facing docs) in the same change. Don't defer documentation to later.
- **Research before implementation** - For new features or unfamiliar APIs, use the research skill at the start of planning to gather context from documentation, real-world code, and community knowledge. Skip this for small, obvious changes.

## Plan Mode

- Make the plan extremely concise. Sacrifice grammar for the sake of concision.
- At the end of each plan, give me a list of unresolved questions to answer, if any.

## Environment and Tooling

### MCP Servers

This project uses MCP (Model Context Protocol) servers to enhance development workflows. **Agents should attempt to use specific skills first and fall back to using MCP servers to save token usage when implementing changes.**

Available MCP servers (run `mcp-cli` to list):

- **context7** - Query documentation for libraries and APIs
- **grep** - Search GitHub for real-world code examples and usage patterns

**When to use MCP servers:**

1. **Before implementing unfamiliar APIs or patterns** - Use `grep/searchGitHub` to find real-world examples
   - Example: `mcp-cli grep/searchGitHub '{"query": ".goreleaser.yaml", "language": "yaml"}'`
   - Search for actual code patterns, not keywords (e.g., "func main()" not "golang tutorial")

2. **When working with external libraries** - Use `context7/query-docs` to look up API documentation
   - First resolve library: `mcp-cli context7/resolve-library-id '{"query": "goreleaser", "libraryName": "goreleaser"}'`
   - Then query docs: `mcp-cli context7/query-docs '{"libraryId": "...", "query": "configuration"}'`

3. **When uncertain about syntax or configuration** - Use grep to see how others solve similar problems
   - Example: Find GitHub Actions workflows, configuration files, or implementation patterns

**Best practices:**

- Use MCP servers early in the implementation process to gather context
- Prefer real-world examples over guessing syntax or patterns
- Search for literal code patterns in grep (like "async function", "import React"), not prose descriptions

**For systematic research during planning:**

- Use the [research skill](.claude/skills/research/SKILL.md) which coordinates context7 and grep searches
- The skill provides structured findings and handles fallbacks when MCP is unavailable

### Worktrees

This project uses [worktrunk (`wt`)](https://github.com/max-sixty/worktrunk) for managing git worktrees, enabling parallel work on multiple branches.

**Core commands:**

```bash
# Create and switch to a new worktree
wt switch -c feat/new-feature

# Switch to an existing worktree
wt switch feat/existing-branch

# List all worktrees with status
wt list

# Merge and clean up a completed worktree
wt merge

# Remove a worktree
wt remove
```

**When to use worktrees:**

- Running multiple AI agents in parallel on different tasks
- Working on a feature while waiting for PR review on another
- Testing changes in isolation without stashing

**Workflow:**

1. Create a worktree: `wt switch -c feat/my-feature`
2. Work on the feature in the new directory
3. When complete, merge back: `wt merge`

See [worktrunk.dev](https://worktrunk.dev) for full documentation.

### Research Skill

When planning new features or making architectural changes, use the [research skill](.claude/skills/research/SKILL.md) to gather context from multiple sources.

**The skill helps you:**

- Query library documentation (context7)
- Find real-world code examples (grep/searchGitHub)
- Understand common patterns and trade-offs
- Make informed implementation decisions

**When to use:**

- Implementing new functionality with unknown patterns
- Working with unfamiliar libraries or APIs
- Evaluating architectural approaches
- Needing real-world examples before deciding

**Research process:**

1. Ask clarifying questions about requirements
2. Request documentation links from the user if they have them
3. Execute multi-phase research (docs → code examples → web if needed)
4. Present structured findings with recommendations
5. Get user approval before implementation

**Important:** Always notify the user if MCP servers are unavailable during research.

## Code Quality & Style

### Pre-commit

Pre-commit hooks (configured in `.pre-commit-config.yaml`) automatically run: fmt, validate, tests, conventional commit validation, trailing whitespace/YAML checks and other lints.

**Common CI failures:**

- Commit message >100 chars or wrong format
- Code not formatted

### Logging and Errors

- Emit detailed, structured logs at key boundaries.
- Make errors explicit and informative.

### Commit Message Requirements

All commits must follow the **conventional commit** pattern. See the [commits skill](.claude/skills/commit/SKILL.md) for detailed guidance.

Quick reference:

```
<type>(<scope>): <description>
```

Common types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

Examples: `feat: add user auth`, `fix: resolve memory leak`, `docs: update install instructions`

### Committing Guidelines

- **NEVER commit directly to main** - Always create a feature branch for your work (e.g., `feat/issue-name` or `fix/issue-name`)
- **Run tests/pre-commit before every commit** - Catches formatting, linting, and prose issues
- **NEVER commit/push without explicit user approval**
- **Avoid hardcoding values that change** - No version numbers, dates, or timestamps in tests. Use runtime checks.
- **When fixing tests** - Understand what's being validated, fix the underlying issue, make expectations flexible
- **Keep summaries brief** - 1-2 sentences, no code samples unless requested

### Pull Request Workflow

When work on a feature branch is complete, follow this workflow to create a pull request:

1. **Ensure branch is up to date** - Rebase on main and resolve any conflicts
2. **Run quality gates** - Execute tests, linters, and builds to ensure all checks pass
3. **Push feature branch** - Push your feature branch to the remote repository with `git push -u origin <branch-name>`
4. **Create pull request** - Use `gh pr create` to open a pull request:
   - Provide a clear title following conventional commit format
   - Include a detailed description explaining why the changes are needed
   - Link to related issues or tickets
   - Example: `gh pr create --title "feat: add user authentication" --body "$(cat <<'EOF'

## Summary

- Implement JWT-based authentication
- Add login and logout endpoints
- Update middleware to verify tokens

## References

- Fixes #123
  EOF
  )"`

5. **Wait for CI checks** - The PR workflow validates the description and runs CI checks
6. **Address review feedback** - Make changes on the feature branch and push to update the PR

**PR Description Requirements:**

- Must include a clear explanation of why changes are needed
- Should reference related issues or tickets
- Cannot be empty or contain only the default template
- The PR workflow will automatically validate and comment if requirements are not met

### Documentation Style Guide

- Be concise. Avoid unnecessary and verbose explanations. Don't bold or emphasize wording.
- Follow the Go [Style Guide](https://google.github.io/styleguide/go/) and [Best Practices](https://google.github.io/styleguide/go/best-practices) docs.
- Avoid common AI writing patterns
- Link to files/documentation appropriately
- No emojis or hype language
- No specific numbers that will change (versions, coverage percentages)
- No line number references
- Review for consistency and accuracy when done

### Comments

Write simple, concise commentary. Only comment on what is not obvious to a skilled programmer by reading the code. Comments should contain proper grammar and punctuation and should be prose-like, rather than choppy partial sentences. A human reading the comments should feel like they are reading a well-written professional paper.

### Zero Narration

Do not narrate actions. Tool calls are structured output - the user sees them directly. Text output wastes context.

Never output:

- Action announcements ("Let me...", "I'll now...", "I'm going to...")
- Summaries of what was done
- Confirmations of success (visible from tool output)
- Explanations of routine operations

Only output text when:

- Asking a question that requires user input
- Reporting an error that blocks progress
- A decision point requires user choice

Otherwise: execute silently.

## Personal preferences

Use these rules to apply my own personal style and preferences to your responses and behaviors.

- Use language specific idioms first and foremost, only overriding with my personal preferences when needed.
- Be concise in your answers. Avoid unnecessary and verbose explanations. Don't summarize what you did, I will ask if I need clarification.
- Attempt to alphabetize things like blocks, functions, variables and other data whenever possible to make readability easier.
- Solve the problem with the simplest approach. I am okay with increased verbosity in order to avoid complexity.
- Avoid clever hacks. I prefer readability and maintainability.
- Favor the use of native tools and standard libraries over third party tools.
- Look for flimsy tests, check for TODOs/stubs when reviewing changes.
- NEVER accept failing tests as "okay" or "acceptable". All tests must pass before declaring success.
- If any test fails, investigate and fix the root cause. No exceptions.
- ALWAYS rebase on main and resolve conflicts before pushing changes to a remote branch.

## Landing the Plane (Session Completion)

**When ending a work session with code changes intended for review**, you MUST complete ALL steps below. Work is NOT complete until the PR is created.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Ensure branch is synced with main** - Rebase on latest main and resolve conflicts:
   ```bash
   git pull --rebase origin main
   ```
5. **PUSH FEATURE BRANCH TO REMOTE** - This is mandatory for changes intended for review:
   ```bash
   git push -u origin <branch-name>
   bd sync
   git status  # Verify push succeeded
   ```
6. **CREATE PULL REQUEST** - Use `gh pr create` with clear title and detailed description
7. **Clean up** - Clear stashes, prune remote branches
8. **Verify** - All changes committed, pushed, and PR created
9. **Hand off** - Provide PR URL and context for next session

**CRITICAL RULES:**

- Work is NOT complete until a PR is created for reviewable changes
- NEVER stop before pushing and creating a PR for reviewable changes
- NEVER say "ready to push when you are" - YOU must push and create the PR
- If push or PR creation fails, resolve and retry until it succeeds

## Findings

- Use the [findings skill](.claude/skills/findings/SKILL.md) at the start of every session.
- Read `./context/findings.md` before any other work.
- Update `./context/findings.md` continuously while you work.
- Log your own mistakes and corrections, not just user feedback.
