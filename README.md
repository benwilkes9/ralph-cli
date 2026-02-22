# Ralph CLI

A Go CLI that runs plan/build loops using [Claude Code](https://docs.anthropic.com/en/docs/claude-code). AKA [Ralph Wiggum loops](https://ghuntley.com/ralph).

## What's a Ralph Wiggum Loop?

A genius technique for long-running agents that don't drift and don't use up the context window. Simply put Claude (or another agent) in a loop and give it the same prompt on every iteration, then marvel at the results. If you want to know more just read [the blog](https://ghuntley.com/ralph).

## Why did I make this CLI?

I made this for myself! The [Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook) is great, but implementing it per repo or project is clunky and time consuming. This CLI is simply a convenience implementation of the Ralph Playbook. I use this instead of the [Anthropic plugin](https://github.com/anthropics/claude-code/tree/main/plugins/ralph-wiggum) because that runs inside Claude, so the context window is an issue (for now!).

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
# 1. Create a feature branch (init/plan/apply won't run on main or master)
git checkout -b my-feature

# 2. Scaffold .ralph/ — interactive prompts will ask about your project
ralph init

# 3. Add your API keys
cp .env.example .env
# Set ANTHROPIC_API_KEY and GITHUB_PAT in .env

# 4. Add at least one .md spec to your specs directory
#    (ralph init creates this for you based on your branch)
specs/my-feature/my-spec-1.md
specs/my-feature/my-spec-2.md

# 5. Run planning loop (generates .ralph/plans/IMPLEMENTATION_PLAN_my-feature.md)
#    Scaffold files are auto-committed on first run
ralph plan

# 6. Run build loop (reads from the implementation plan, implements tasks one at a time)
ralph apply

# 7. Check progress
ralph status
```

## IMPORTANT Guidelines & Considerations

### Specs
The first fundamental idea is that you can implement a full feature in one shot in a single code repo (monolith, monorepo, microservice, whatever — a repo). You need well-written, clear, unambiguous specs. Think context engineering, spec-driven development — the results you get from Claude or any LLM will depend on the context you give it. SPEND TIME ON THIS.

### Prompts
`ralph init` generates two prompt files: `.ralph/prompts/plan.md` and `.ralph/prompts/build.md`. These are the instructions that get fed to Claude on every iteration of the plan and build loops respectively. They're yours to customise — tweak them to suit your project, your conventions, your workflow. The defaults are a solid starting point but you'll get better results by tailoring them. SPEND TIME ON THIS.

### Guardrails / Backpressure
The other fundamental idea is "backpressure" or "guardrails" — you need to give your build agents clear parameters and guidance. Automated testing, linting, security checking, etc. This is where you need to put your engineering hat on. If you're going to expect Claude to implement good quality code consistently, you need to tell it what good looks like. SPEND TIME ON THIS.

### Token Use
If you're coming from the "human in the loop" approach to agentic engineering — one task at a time — this will use a lot more tokens. And if you don't give it well-written, clear, unambiguous specs and clear guardrails then you will waste a lot of tokens! YOU HAVE BEEN WARNED!

### Monitoring
The CLI gives you well-formatted output of what's going on — thinking, tool use, token use, results. It pays to monitor it closely, at least for the first few iterations.

I think of all this as an implementation of the [four foundational agentic patterns](https://www.nibzard.com/agentic-handbook#foundational-patterns-you-can-use-immediately): plan then execute; inversion of control; reflection loop; action trace monitoring & interruption.

### Branch-Aware Conventions

Ralph is branch-aware — plans and specs are isolated per branch so parallel features don't collide:

- **All commands** (`init`, `plan`, `apply`) **must be run on a feature branch** — they'll error on `main` or `master`
- **Specs directory** is chosen during `ralph init` (default: `specs/`), with the branch appended automatically (e.g. `specs/my-feature/`). Overridable per-run with `--specs`
- **Plans** are stored at `.ralph/plans/IMPLEMENTATION_PLAN_{branch}.md` (e.g. `IMPLEMENTATION_PLAN_my-feature.md`)
- `ralph plan` **requires at least one `.md` spec** in the specs directory before it will run
- `ralph apply` **requires an implementation plan** — run `ralph plan` first
- `ralph status` automatically reads the plan for your current branch
- **Auto-commit** — scaffold files (`.ralph/`, `AGENTS.md`, `.env.example`, `.gitignore`) are auto-committed on the first `ralph plan` run if not already tracked

## Commands

| Command | Description |
|---------|-------------|
| `ralph init` | Scaffold `.ralph/` in current repo (must be on a feature branch) |
| `ralph plan` | Run planning loop (generates implementation plan from specs) |
| `ralph apply` | Run build loop (implements tasks from the plan one at a time) |
| `ralph status` | Progress summary — tasks done, costs, pass/fail |

### Flags

| Flag | Description |
|------|-------------|
| `-n, --max <N>` | Limit iterations (e.g. `ralph plan -n 3`) |
| `--specs <dir>` | Use a custom specs directory (default: `specs/{branch}`) |

Flags can be combined: `ralph plan -n 3 --specs specs/custom-dir`

## Container Isolation

Ralph runs Claude Code inside a Docker container with a bind-mounted workspace. Changes made by the agent appear on the host filesystem in real time — no sync step required.

### Network Firewall

Outbound network access is restricted to an allowlist of domains via iptables rules configured at container startup. All other outbound traffic is dropped.

**Default allowlist** (always included): `api.anthropic.com`, `github.com`, `api.github.com`, `registry.npmjs.org`

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

## Configuration

`ralph init` detects your project ecosystem and asks interactive questions about your run command, project goal, and specs directory. The generated `.ralph/config.yaml` can be further customised:

- Project name and agent
- Backpressure commands (test, typecheck, lint)
- Phase-specific settings (prompt files, max iterations)
- Network allowlist (extra domains the container can reach)
- Dependency directory for volume caching (e.g. `node_modules`, `.venv`)

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
```

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
