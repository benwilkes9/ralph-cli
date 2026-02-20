# Ralph CLI

A Go CLI that orchestrates autonomous plan/build iteration loops using [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Replaces a bash-based workflow with a single binary.

Based on the [Ralph Wiggam loop](https://ghuntley.com/ralph) and the [Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook).

## Installation

### Homebrew (macOS/Linux)

```bash
brew install benwilkes9/tap/ralph
```

### Curl script

```bash
curl -fsSL https://raw.githubusercontent.com/benwilkes9/ralph-cli/main/install.sh | sh
```

### Go install

```bash
go install github.com/benwilkes9/ralph-cli/cmd/ralph@latest
```

## Quick Start

```bash
# 1. Scaffold .ralph/ in your repo
ralph init

# 2. Add your API keys
cp .env.example .env
$EDITOR .env

# 3. Run planning loop (generates .ralph/IMPLEMENTATION_PLAN.md)
ralph plan

# 4. Run build loop (implements tasks one at a time)
ralph apply

# 5. Check progress
ralph status
```

## Commands

| Command | Description |
|---------|-------------|
| `ralph init` | Scaffold `.ralph/` in current repo |
| `ralph plan` | Run planning loop (generates implementation plan) |
| `ralph apply` | Run build loop (implements tasks one at a time) |
| `ralph status` | Progress summary — tasks done, costs, pass/fail |

### Flags

- `ralph plan -n 3` — limit planning to 3 iterations
- `ralph apply -n 10` — limit build to 10 iterations

## Configuration

After `ralph init`, edit `.ralph/config.yaml` to configure:

- Project name and agent
- Backpressure commands (test, typecheck, lint)
- Phase-specific settings (prompt files, max iterations)

See `ralph init` output for file locations.

## Development

### Prerequisites

- Go 1.25+
- `~/go/bin` on your `$PATH` (add `export PATH="$HOME/go/bin:$PATH"` to `~/.zshrc`)

### Setup

```bash
# Install dev tools
brew install golangci-lint
go install github.com/zricethezav/gitleaks/v8@latest

# Install git hooks
go tool lefthook install
```

### Build & Test

```bash
make build    # Build to bin/ralph
make test     # Run tests
make lint     # Run linter
make install  # Install to $GOPATH/bin
```

## License

[MIT](LICENSE)
