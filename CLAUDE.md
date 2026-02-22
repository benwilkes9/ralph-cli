# CLAUDE.md

## Project

Ralph CLI — a Go CLI that orchestrates autonomous plan/build iteration loops using Claude Code.

## Commands

```bash
ralph init    # Scaffold .ralph/ in current repo
ralph plan    # Run planning loop (generates .ralph/IMPLEMENTATION_PLAN.md)
ralph apply   # Run build loop (implements tasks one at a time)
ralph status  # Progress summary — tasks done, costs, pass/fail
```

## Build & Test

```bash
make build    # Build to bin/ralph
make install  # Install to $GOPATH/bin
make test     # go test -race ./... with coverage report
make cover    # open HTML coverage report in browser
make lint     # golangci-lint run ./...
```

## Pre-commit Hooks

Run before committing (managed by lefthook):

```bash
go tool lefthook run pre-commit
```

This runs 5 checks in parallel:
- `golangci-lint run --fix ./...` — lint + auto-fix (30+ linters)
- `make test` — all tests with `-race` and goroutine leak detection (goleak)
- `nilaway ./...` — interprocedural nil pointer analysis
- `govulncheck ./...` — dependency vulnerability scanning
- `gitleaks protect --staged` — secret detection

Always run this before committing code.

## CI Pipeline

GitHub Actions runs 3 parallel jobs on every push and PR:
- **lint** — golangci-lint, nilaway
- **test** — `go test -race` with coverage (uploaded to Codecov), build
- **security** — govulncheck, gitleaks

## Releases

Tag-triggered via goreleaser (`git tag v0.1.0 && git push origin v0.1.0`):
- Builds linux/darwin x amd64/arm64 binaries
- Cosign signs checksums (keyless via GitHub OIDC)
- Syft generates SBOMs (SPDX-JSON)
- SLSA Level 3 provenance via slsa-github-generator

## Architecture

```
cmd/ralph/main.go       — Cobra CLI entrypoint, all subcommands defined here
internal/config/        — .ralph/config.yaml parsing + defaults
internal/stream/        — JSONL stream parser, ANSI formatter, stats tracking
internal/loop/          — Iteration loop orchestrator, stale detection
internal/git/           — Git operations (shelling out to git CLI)
internal/log/           — JSONL log file tee writer
internal/docker/        — Docker build + run (shelling out to docker CLI)
internal/summary/       — Final summary box rendering
templates/              — Embedded scaffold templates for ralph init (//go:embed)
```

## Key Decisions

- **Shell out** to `git` and `docker` CLIs rather than using Go SDKs — simpler, fewer deps
- **No Docker SDK** — `docker build` and `docker run` via exec.Command
- **stream-json format** — Claude's `--output-format=stream-json` produces JSONL; we parse line-by-line with bufio.Scanner + json.Unmarshal
- **Embedded templates** — scaffold files use Go's `text/template` + `//go:embed`
- **Env var allowlist** — `.env` loading only permits `ANTHROPIC_API_KEY` and `GITHUB_PAT`; update `allowedEnvVars` in `internal/docker/docker.go` when adding new vars
- **Version sanitization** — language versions detected from repo files are validated against `safeVersion` regex before template interpolation to prevent shell injection

## Pre-commit Workflow

Before committing, always:

1. Run `go tool lefthook run pre-commit`
2. Ask the user if they want to run `/security-review` before committing.

## Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation only
- `chore:` build, CI, tooling
- `refactor:` code change that neither fixes a bug nor adds a feature
- `test:` adding or updating tests

Use scope when relevant: `feat(stream): add color support`
Single line only. No body, no `Co-Authored-By` trailer.

## Testing Conventions

- Test behaviour, not implementation — tests should survive internal refactoring
- Use table-driven tests for input/output variance; group integration-style assertions by concern
- Test at the package boundary via exported functions; don't reach into unexported internals
- Prioritise business logic, edge cases, and error paths (`error` returns are cheap to test in Go)
- Target ~80% coverage per business logic package; exclude CLI entrypoints and thin adapters
- Use `testify/assert` for assertions, `testify/require` to fail fast on preconditions
- Prefer real temp dirs (`t.TempDir()`) over filesystem mocks
- Fake the process boundary (git, docker) via injected interfaces, not by mocking internals

### Command-level integration tests (`cmd/ralph`)

Tests start at `cmd.Execute()` — the same entry point as typing `ralph plan` — and run all
application code up to but not including the external process boundary (Docker). This is the
equivalent of REST API tests that mock the database and call the real HTTP handler.

**Pattern:**
1. Create a real temporary git repo (`initRepoWithConfig`) and `os.Chdir` into it
2. Inject a `fakeOrchestrator` in place of `docker.BuildAndRun`
3. Call `cmd.Execute()` and assert on filesystem side-effects and `fake.calls`

**Key seams:**
- `Orchestrator` interface — injected into `planCmd`/`applyCmd`; production uses `realOrchestrator{}`
- `cmd.InOrStdin()` / `cmd.OutOrStdout()` — used in `initCmd`/`statusCmd`; tests redirect with `cmd.SetIn`/`cmd.SetOut`
- Git subprocesses — use real `git` CLI in a temp repo; do **not** mock git at this layer
- Do **not** use `t.Parallel()` in `cmd/ralph` tests — `os.Chdir` is process-global

## Conventions

- Keep packages focused — one responsibility per package
- Prefer returning errors over panicking
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Version injected via `-ldflags` at build time from git tags
- Test files live alongside source: `foo_test.go` next to `foo.go`
