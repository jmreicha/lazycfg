# Iteration Instructions

You are working on a beads issue in an autonomous iteration loop.

## Your Task

Complete the issue using the following the project guidelines and steps.

## Process

1. **Verify issue is actionable**

- Check if issue status is blocked: The issue details will show if status is "blocked"
- If status is blocked, immediately exit with message: "Issue is blocked and requires manual review. Skipping autonomous execution."
- Do NOT attempt to work on blocked issues - they need manual intervention

2. **Check for existing work**

- Check if PR already exists for this issue: `gh pr list --search "in:title issue-name"`
- If PR exists, check its status and review state: `gh pr view <pr-number> --json reviewDecision,reviews`
- If review state is "CHANGES_REQUESTED":
  - Check review comments: `gh pr view <pr-number> --comments`
  - Address each requested change
  - Make the necessary code changes
  - Push changes to update the PR
  - Optionally reply to comments when addressed: `gh pr comment <pr-number> --body "Addressed in <commit-sha>"`
- Review failing checks and error messages
- If checks are failing, fix issues in a separate commit and push to update the PR
- If no PR exists, check if branch exists: `git branch -a | grep feat/issue-name`
- If branch exists remotely, checkout and continue: `git checkout feat/issue-name`

3. **Create a branch (if needed)**

- Check current branch: `git branch`
- If on main and no feature branch exists, create one: `git checkout -b feat/issue-name` or `git checkout -b fix/issue-name`
- If already on a feature branch for this issue, continue working on it
- NEVER work directly on main

4. **Understand the requirement**

- Run 'bd show $bead' to understand the issue
- Read the issue details carefully
- Check acceptance criteria if present
- Review any referenced files or context

5. **Review**

- Use any relevant MCP servers to understand documentation, code, etc.
- Read the "Recent Learnings" section below
- Investigate the codebase to find the root cause
- Avoid repeating past mistakes
- Apply successful patterns from history
- If you have already identified the target files, stop reading and start editing.

6. **Make changes**

- Follow coding standards in AGENTS.md
- Make small, atomic commits
- Write clear conventional commit messages
- After the first edit, immediately run `task check`; if it fails, fix and rerun. Do not resume exploration.

7. **Test your changes**

- Create unit tests for your fix (create any test files if needed)
- Create any integration tests if applicable
- Run checks: `task check` (runs fmt, lint, and test)
- Verify no regressions
- Fix any and all failures before continuing

8. **Report findings** (optional)

- If you discover important learnings, wrap them in tags:
- `<findings>Your learning here</findings>`
- Be specific and actionable
- Add the findings to the `## Findings` section in AGENTS.md

9. **Complete and push**

- Stage ONLY your code changes (NOT .beads/): `git restore .beads/`
- Ensure branch is synced with main: `git pull --rebase origin main`
- Push feature branch: `git push -u origin <branch-name>`
- Create or update pull request: `gh pr create --title "type: description" --body "Detailed explanation"`
- Add the `vibes` label to the PR: `gh pr edit <pr-number> --add-label vibes`
- Check PR status: `gh pr checks` (do NOT use --watch flag as tests can take several minutes)
- If checks fail, fix issues in a separate commit and push to update PR
- When task is complete, output: `<promise>COMPLETE</promise>`
- Update issue status if appropriate

## Guidelines

- IMMEDIATELY check if issue status is "blocked" at the start - if blocked, exit with message and do NOT proceed
- ALWAYS check for existing PR first before creating a new branch
- If PR review state is "CHANGES_REQUESTED", address ALL requested changes before continuing
- If PR review state is "APPROVED" or "COMMENTED" (without changes requested), no action needed on comments
- NEVER commit to main directly. Always work on a feature branch
- Do NOT include .beads/ in your commit - run 'git restore .beads/' before staging
- Do NOT run 'bd close' - the script handles closing the bead after PR is created
- MUST push feature branch and create PR before signaling completion
- MUST verify all PR checks pass before signaling completion
- NEVER use 'gh pr checks --watch' as it blocks indefinitely - use 'gh pr checks' without --watch
- If PR checks fail, fix issues in a separate commit (not amend)
- PR description must explain why changes are needed (not just what changed)
- All lint and test violations MUST be fixed before committing
- Write meaningful tests that cover the changes
- When addressing review comments, make changes in separate commits (not amend) and push to update the PR

See AGENTS.md for:

- Code quality standards
- Commit message format
- Testing requirements
- Documentation style

Begin your work now.
