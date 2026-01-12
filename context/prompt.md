# Iteration Instructions

You are working on a beads issue in an autonomous iteration loop.

## Your Task

Complete the issue shown below following the project guidelines.

## Process

1. **Check for existing PR**

- Check if PR already exists for this issue: `gh pr list --search "in:title issue-name"`
- If PR exists, check its status: `gh pr view <pr-number> --json statusCheckRollup`
- Review failing checks and error messages
- If checks are failing, fix issues in a separate commit and push to update the PR
- If no PR exists, proceed to create a branch

2. **Create a branch**

- Check current branch: `git branch`
- If on main, create feature branch: `git checkout -b feat/issue-name` or `git checkout -b fix/issue-name`
- If PR exists, checkout existing branch: `git checkout <branch-name>`
- NEVER work directly on main

3. **Understand the requirement**

- Run 'bd show $bead' to understand the issue
- Read the issue details carefully
- Check acceptance criteria if present
- Review any referenced files or context

4. **Review**

- Use any relevant MCP servers to understand documentation, code, etc.
- Read the "Recent Learnings" section below
- Investigate the codebase to find the root cause
- Avoid repeating past mistakes
- Apply successful patterns from history

5. **Make changes**

- Follow coding standards in AGENTS.md
- Make small, atomic commits
- Write clear conventional commit messages

6. **Test your changes**

- Create unit tests for your fix (create any test files if needed)
- Create any integration tests if applicable
- Run checks: `task check` (runs fmt, lint, and test)
- Verify no regressions
- Fix any and all failures before continuing

7. **Report findings** (optional)

- If you discover important learnings, add them to the `## Findings` section in AGENTS.md
- Be specific and actionable

8. **Complete and push**

- Stage ONLY your code changes (NOT .beads/): `git restore .beads/`
- Ensure branch is synced with main: `git pull --rebase origin main`
- Push feature branch: `git push -u origin <branch-name>`
- Sync beads: `bd sync`
- Create or update pull request: `gh pr create --title "type: description" --body "Detailed explanation"`
- Verify all PR checks pass: `gh pr checks`
- If checks fail, fix issues in a separate commit and push to update PR
- When task is complete, output: `<promise>COMPLETE</promise>`
- Update issue status if appropriate

## Guidelines

- ALWAYS check for existing PR first before creating a new branch
- NEVER commit to main directly. Always work on a feature branch
- Do NOT include .beads/ in your commit - run 'git restore .beads/' before staging
- Do NOT run 'bd close' - the script handles closing the bead after PR is created
- MUST push feature branch and create PR before signaling completion
- MUST verify all PR checks pass before signaling completion
- If PR checks fail, fix issues in a separate commit (not amend)
- PR description must explain why changes are needed (not just what changed)
- All lint and test violations MUST be fixed before committing
- Write meaningful tests that cover the changes

See AGENTS.md for:

- Code quality standards
- Commit message format
- Testing requirements
- Documentation style

Begin your work now.
