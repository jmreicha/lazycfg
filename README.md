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
- **Task** - [Install Task](https://taskfile.dev/installation/)
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

2. Check required tools:

```bash
task check-tools
```

3. Bootstrap dependencies and tools:

```bash
task bootstrap
```

4. Install beads (issue tracker) if not already installed:

```bash
# macOS/Linux
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Or with Homebrew
brew install steveyegge/beads/bd
```

5. Install pre-commit hooks:

```bash
prek install
# Or with pre-commit
# pre-commit install
```

6. Verify your setup:

```bash
# Check all tools at once
task check-tools

# Or check individually
go version
task --version
bd version
pre-commit --version
```

### Working with Beads

This project uses [beads](https://github.com/steveyegge/beads) for local issue tracking. Beads is a git-backed, distributed issue tracker optimized for AI agents.

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
5. Run checks: `task check` (runs fmt, lint, and test)
6. Commit with conventional commit format: `git commit -m "feat: your feature"`
7. Sync beads: `bd sync`
8. Push changes: `git push`

### Available Tasks

View all available tasks with:

```bash
task --list
```

Common tasks:

```bash
task check-tools     # Check if required CLI tools are installed
task bootstrap       # Bootstrap and set up dependencies and tools
task build           # Build the binary
task fmt             # Format Go code
task lint            # Run linters
task test            # Run tests
task check           # Run fmt, lint, and test
task clean           # Clean build artifacts
```

### Autonomous Iteration

For autonomous development, use the iteration loop to work through beads issues:

```bash
# Run with default settings (max 50 iterations)
./script/iterate.sh

# Custom max iterations
./script/iterate.sh --max-iterations 100

# Work on specific issue
./script/iterate.sh --issue-id <issue-id>

# With environment variable
MAX_ITERATIONS=25 ./script/iterate.sh
```

The loop:

- Pulls ready tasks from beads
- Shows issue details and historical learnings from `context/progress.txt`
- Records improvements after each session
- Updates beads issue status
- Continues until complete or max iterations reached
