# Ralph CLI

A CLI implementation of the Ralph Wiggam loop, made famous in the [Geoff Huntley Blog](https://ghuntley.com/ralph). Also based on the [Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook)

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

