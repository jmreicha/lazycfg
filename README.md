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
- **golangci-lint** - Go linter
- **pre-commit** - Pre-commit hook framework
- **bd (beads)** - Issue tracker for git
- **common-repo** - Repository configuration management
- **prek** (optional) - Pre-commit hook management tool
- **mcp-cli** (optional) - MCP server interface

### Installation

1. Clone the repository:

```bash
git clone git@github.com:jmreicha/lazycfg.git
cd lazycfg
```

2. Bootstrap the development environment:

```bash
# One command to install everything
task bootstrap
```

This will:

- Check for required tools (go, golangci-lint, bd, pre-commit)
- Auto-install missing tools (uses Homebrew on macOS)
- Install pre-commit hooks (via prek or pre-commit)
- Download and verify Go dependencies

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
task install-tools   # Install missing tools and pre-commit hooks
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
# Run with default settings (picks the first open issue)
task iterate

# Custom max iterations
task iterate -- --max-iterations 10

# Work on specific issue
task iterate -- --issue-id <issue-id>

# With environment variable
MAX_ITERATIONS=25 task iterate

# Use different AI agent
AI_AGENT='claude' task iterate

# Adjust timeout (default 30 minutes)
OPENCODE_TIMEOUT=3600 task iterate
```

The loop:

- Pulls ready tasks from beads
- Shows issue details and historical learnings from `context/progress.txt`
- Records improvements after each session
- Updates beads issue status
- Continues until complete or max iterations reached
