# lazycfg

A command line tool to simplify the creation and management of complicated configurations.

## Why?

Imagine you are a new hire at a company and you need to set up your local
environment, but the setup process is complicated and involves multiple steps,
and everybody is busy. You need to install various tools, configure them, set up
your environment variables, etc. You also need to get connected to you cloud
environments, set Kubernetes configuration, and deal with other complicated
configs.

This setup process can be time-consuming and error-prone. Additionally, you may
not understand (or care about) security implications or know about advanced
configuration which you may want but don't know exist.

`lazycfg` aims to simplify all of this setup by providing a simple command line
interface to handle this for you so you can focus on more important things.

## Getting Started

### Prerequisites

This project requires the following tools:

- **Go** 1.23+ - [Install Go](https://go.dev/doc/install)
- **pre-commit** - [Install pre-commit](https://pre-commit.com/#install)
- **bd (beads)** - Issue tracker for git
- **prek** - Configuration management tool
- **mcp-cli** - MCP server interface
- **common-repo** - Repository tooling

### Installation

1. Clone the repository:

```bash
git clone git@github.com:jmreicha/lazycfg.git
cd lazycfg
```

2. Install Go dependencies:

```bash
go mod download
```

3. Install beads (issue tracker):

```bash
# macOS/Linux
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Or with Homebrew
brew install steveyegge/beads/bd
```

4. Install pre-commit hooks:

```bash
pre-commit install
```

5. Verify your setup:

```bash
# Check Go installation
go version

# Check beads installation
bd version

# Check pre-commit
pre-commit --version
```

### Working with Beads

This project uses [beads](https://github.com/steveyegge/beads) for issue tracking. Beads is a git-backed, distributed issue tracker optimized for AI agents.

**Essential commands:**

```bash
# View ready work (no blockers)
bd ready

# Create a new issue
bd create "Task title" -p 1 -d "Description"

# Show issue details
bd show <issue-id>

# Update issue status
bd update <issue-id> --status in_progress

# Close completed issue
bd close <issue-id>

# Sync before pushing
bd sync
```

### Development Workflow

1. Check for ready work: `bd ready`
2. Create or claim an issue
3. Create a feature branch: `git checkout -b feature/your-feature`
4. Make your changes
5. Run tests: `go test ./...`
6. Commit with conventional commit format: `git commit -m "feat: your feature"`
7. Sync beads: `bd sync`
8. Push changes: `git push`

### Commit Message Format

This project uses conventional commits:

```
<type>(<scope>): <description>

Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore
```

Examples:
- `feat: add user authentication`
- `fix: resolve memory leak in config parser`
- `docs: update installation instructions`

## Contributing

See [AGENTS.md](AGENTS.md) for detailed development guidelines and tooling information.
