# Iteration Instructions

You are working on a beads issue in an autonomous iteration loop.

## Your Task

Complete the issue shown below following the project guidelines.

## Process

1. **Understand the requirement**

- Run 'bd show $bead' to understand the issue
- Read the issue details carefully
- Check acceptance criteria if present
- Review any referenced files or context

2. **Review learnings**

- Read the "Recent Learnings" section below
- Investigate the codebase to find the root cause
- Avoid repeating past mistakes
- Apply successful patterns from history

3. **Make changes**
   - Follow coding standards in AGENTS.md
   - Make small, atomic commits
   - Write clear conventional commit messages

4. **Test your changes**

- Create unit tests for your fix (create any test files if needed)
- Create any integration tests if applicable
- Run lints and other validations to check for issues
- Run tests: `go test ./...`
- Verify no regressions
- Fix any and all failures before continuing

5. **Report findings** (optional)

- If you discover important learnings, wrap them in tags:
- `<findings>Your learning here</findings>`
- Be specific and actionable

6. **Signal completion**

- Stage ONLY your code changes (NOT .beads/):
- DO NOT push or create PR - the script will handle this
- When task is complete, output: `<promise>COMPLETE</promise>`
- Update issue status if appropriate

## Guidelines

- Do NOT commit to main directly
- Do NOT include .beads/ in your commit - run 'git restore .beads/' before staging
- Do NOT push or create PR - the script handles git push and gh pr create
- Do NOT run 'bd close' - the script handles closing the bead after PR is created
- All lint and test violations MUST be fixed before committing
- Write meaningful tests that cover the changes

See AGENTS.md for:

- Code quality standards
- Commit message format
- Testing requirements
- Documentation style

Begin your work now.
