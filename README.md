# Ralph CLI

A Go CLI that runs [Claude Code](https://docs.anthropic.com/en/docs/claude-code) in a loop, AKA [Ralph Wiggum loops](https://ghuntley.com/ralph).

## Motivation

The [Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook) is great, but implementing it per repo or project is clunky and time-consuming. This CLI is an opinionated convenience implementation of the Ralph Playbook. It runs outside of Claude in a [sandboxed container](#container-isolation), so there are no context window constraints — unlike the [Anthropic plugin](https://github.com/anthropics/claude-code/tree/main/plugins/ralph-wiggum) which runs inside Claude.

## Installation

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
# 1. Create a feature branch (init/plan/build won't run on main or master)
git checkout -b my-feature

# 2. Scaffold .ralph/ — interactive prompts will ask about your project
ralph init

# 3. Add your credentials
cp .env.example .env
# Set ANTHROPIC_API_KEY (or CLAUDE_CODE_OAUTH_TOKEN) and GITHUB_PAT in .env

# 4. Add at least one .md spec to your specs directory
#    (ralph init creates this for you based on your branch)
specs/my-feature/my-spec-1.md
specs/my-feature/my-spec-2.md

# 5. Run planning loop (generates .ralph/plans/IMPLEMENTATION_PLAN_my-feature.md)
#    Scaffold files are auto-committed on first run
ralph plan

# 6. Run build loop (reads from the implementation plan, implements tasks one at a time)
ralph build

# 7. Check progress
ralph status
```

## Authentication

Ralph CLI supports two authentication methods for Claude Code:

| Method | Env Var | Billing |
|--------|---------|---------|
| API Key | `ANTHROPIC_API_KEY` | Usage-based (API billing) |
| OAuth Token | `CLAUDE_CODE_OAUTH_TOKEN` | Claude Max subscription |

Set **one** in your `.env` file. If both are set, the API key takes precedence.

### Using an API Key (default)

Set `ANTHROPIC_API_KEY` in `.env`. This uses standard Anthropic API billing.

### Using Claude Max (OAuth)

If you have a Claude Max subscription, you can use your subscription instead of API billing:

1. Run `claude setup-token` to obtain an OAuth token (`sk-ant-oat01-*` format)
2. Set `CLAUDE_CODE_OAUTH_TOKEN` in `.env`
3. Leave `ANTHROPIC_API_KEY` blank or remove it

> **Upgrading existing repos:** Run `ralph init --force` to update scaffold files with OAuth support.

## Commands

| Command | Description |
|---------|-------------|
| `ralph init` | Scaffold `.ralph/` in current repo (must be on a feature branch). Use `--force` to overwrite existing files |
| `ralph plan` | Run planning loop (generates implementation plan from specs) |
| `ralph build` | Run build loop (implements tasks from the plan one at a time) |
| `ralph status` | Progress summary — tasks done, costs, pass/fail |

### Flags

| Flag | Description |
|------|-------------|
| `-n, --max <N>` | Limit iterations (e.g. `ralph plan -n 3`) |
| `--specs <dir>` | Override the specs directory configured in `.ralph/config.yaml` |
| `--force` | Overwrite existing scaffold files (useful after upgrading ralph) |

Flags can be combined: `ralph plan -n 3 --specs specs/custom-dir`

## Configuration

`ralph init` detects your project ecosystem and asks interactive questions about your run command, project goal, and specs directory. The generated `.ralph/config.yaml` can be further customised:

- Project name and agent
- Backpressure commands (test, typecheck, lint)
- Phase-specific settings (prompt files, max iterations)
- Network allowlist (extra domains the container can reach)
- Dependency directory for volume caching (e.g. `node_modules`, `.venv`)
- Additional directories for [multi-repo support](#multi-repo-support)

```yaml
# Auto-populated by ralph init based on detected ecosystem.
# Add more domains as needed.
network:
  extra_allowed_domains:
    - pypi.org
    - files.pythonhosted.org

# Cache dependency directory in a named Docker volume to survive rebuilds
docker:
  deps_dir: .venv

# Multi-repo support — coordinate changes across multiple repositories
additional_directories:
  - /Users/you/code/repo-b
  - /Users/you/code/repo-c
```

## Branch Isolation

Ralph is branch-aware — plans and specs are isolated per branch so parallel features don't collide:

- **All commands** (`init`, `plan`, `build`) **must be run on a feature branch** — they'll error on `main` or `master`
- **Specs directory** is chosen during `ralph init`. Preset options (e.g. `specs/`) have the branch appended automatically (e.g. `specs/my-feature/`). Custom paths are used as-is. Overridable per-run with `--specs`
- **Plans** are stored at `.ralph/plans/IMPLEMENTATION_PLAN_{branch}.md` (e.g. `IMPLEMENTATION_PLAN_my-feature.md`)

## Multi-Repo Support

Ralph can orchestrate changes across multiple repositories in a single loop. This is useful for coordinated changes across microservices, split frontend/backend repos, etc.

Add `additional_directories` to your `.ralph/config.yaml` with absolute paths to each additional repo:

```yaml
additional_directories:
  - /Users/you/code/api-service
  - /Users/you/code/shared-lib
```

**Requirements:**
- All paths must be absolute
- Each path must be a git repository
- All repos (primary + additional) must be on the **same branch** — Ralph errors during preflight if any repo is on the wrong branch
- No two directories can share the same basename (they're mounted at `/workspace/{basename}`)

**How it works:**
- Each additional repo is bind-mounted into the container alongside the primary repo
- Claude Code receives `--add-dir` flags so it can read and write across all repos
- Stale detection tracks HEAD across all repos — any repo changing resets the stale counter
- After each iteration, Ralph pushes all repos that have new commits

## Container Isolation

Ralph runs Claude Code inside a Docker container with a bind-mounted workspace. Changes made by the agent appear on the host filesystem in real time — no sync step required.

### Network Firewall

Outbound network access is restricted to an allowlist of domains via iptables rules configured at container startup. All other outbound traffic is dropped.

**Default allowlist** (always included): `api.anthropic.com`, `claude.ai`, `github.com`, `api.github.com`, `registry.npmjs.org`

`ralph init` automatically adds the package registry domains for your detected ecosystem:

| Ecosystem | Extra Domains |
|---|---|
| Python (uv, poetry) | `pypi.org`, `files.pythonhosted.org` |
| Go | `proxy.golang.org`, `sum.golang.org`, `storage.googleapis.com` |
| Rust (cargo) | `crates.io`, `static.crates.io`, `index.crates.io` |
| Node (npm, yarn, pnpm) | _(covered by default allowlist)_ |

You can add more domains in `.ralph/config.yaml` under `network.extra_allowed_domains`.

### Security Layers

| Layer | Threat Mitigated |
|---|---|
| Network firewall (iptables) | Data exfiltration to arbitrary hosts |
| Non-root user (`runuser`) | Privilege escalation, firewall tampering |
| `no-new-privileges` | Setuid/capability escalation |
| Env var allowlist | Injection via compromised `.env` |
| Bind mount scoping | Access to files outside project |

### Recovery

If the agent corrupts workspace files, use `git checkout` or `git stash` to recover — the bind mount means git operates on the same files.

## Important Practices

### Specs
You can implement a full feature in one shot — in a single repo or across [multiple repos](#multi-repo-support). For this to work **you must provide well-written, clear, unambiguous specs**. Think context engineering, spec-driven development — the results you get will depend on the context you give it.

### Prompts
`ralph init` generates two prompt files: `.ralph/prompts/plan.md` and `.ralph/prompts/build.md`. These are the instructions that get fed to Claude on every iteration of the plan and build loops respectively. They're yours to customise — tweak them to suit your project, your conventions, your workflow. The defaults are just a solid starting point. Again, this is crucial context.

### Guardrails / Backpressure
**You must give your build agents clear parameters and guidance**. Automated deterministic guardrails like testing, linting, security checking, etc. You need to do this for precommit hooks (as well as CI) so that the agent will review and fix before committing on each iteration.

There is also the `AGENTS.md`; this is also prepopulated depending on your tech stack, but again you need to review it and tailor it to your conventions, including the validation steps you expect the agent to run, which obviously needs to align with your precommit hooks.

This is where you need to put your engineering hat on. If you're going to expect Claude to implement good quality code consistently, you need to tell it what good looks like and put the guardrails in place.

### Token Usage
If you're coming from the "human in the loop" approach to agentic engineering — one task at a time — this will use a lot more tokens. And if you don't give it well-written, clear, unambiguous specs and clear guardrails then you will **waste a lot of tokens! You have been warned!!**

### Monitoring
The CLI gives you well-formatted output of what's going on — thinking, tool use, token use, results. It pays to monitor it closely, at least for the first few iterations.

All of the above is an implementation of the [four foundational agentic patterns](https://www.nibzard.com/agentic-handbook#foundational-patterns-you-can-use-immediately): plan then execute; inversion of control; reflection loop; action trace monitoring & interruption. Running in a loop is not a silver bullet — it needs engineering.

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

### Testing Local Changes in Docker

Ralph's Docker container installs the published `ralph@latest` from the Go module proxy. To test unpublished changes inside the container, cross-compile a Linux binary and place it next to your host binary — `findLinuxBinary()` will automatically mount it into the container:

```bash
# Build and install host binary + cross-compile for container
make install && GOOS=linux GOARCH=arm64 go build -o "$(go env GOPATH)/bin/ralph-linux" ./cmd/ralph/
```

> Use `GOARCH=amd64` if your Docker VM runs x86_64 (check with `docker info --format '{{.Architecture}}'`).

If you also need a fresh Claude CLI (e.g. testing new flags), rebuild the Docker image without cache from the target repo:

```bash
docker build --no-cache -t ralph-loop:latest -f .ralph/docker/Dockerfile .
```

## License

[MIT](LICENSE)
