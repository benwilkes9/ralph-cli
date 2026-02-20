# Contributing to Ralph CLI

## Reporting Bugs & Requesting Features

Use the [issue tracker](https://github.com/benwilkes9/ralph-cli/issues). Please check existing issues first.

## Development Setup

Requires Go 1.25+ with `~/go/bin` on your `$PATH`.

```bash
brew install golangci-lint
go install github.com/zricethezav/gitleaks/v8@latest
git clone https://github.com/benwilkes9/ralph-cli.git
cd ralph-cli
go tool lefthook install
make build
```

## Build & Test

```bash
make build    # Build to bin/ralph
make test     # Run tests with -race
make lint     # Run golangci-lint
```

## Submitting Changes

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Run `go tool lefthook run pre-commit` and fix any issues
4. Commit using [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) — single line, no body
5. Open a pull request against `main`

Keep PRs focused, describe what/why/how, and ensure CI passes.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
