# Contributing to Plandex

Thanks for your interest in contributing. Plandex is a terminal-based AI coding agent — there's plenty to do.

## Setup

```bash
git clone https://github.com/plandex-ai/plandex
cd plandex
```

Plandex consists of three Go modules under `app/`:

| Module | Path | Description |
|--------|------|-------------|
| `plandex-cli` | `app/cli/` | Terminal UI, REPL, plan execution |
| `plandex-server` | `app/server/` | API server, LLM orchestration, DB |
| `plandex-shared` | `app/shared/` | Shared types, models, utilities |

The CLI and server both depend on `shared` via a local `replace` directive in their `go.mod`:

```
replace plandex-shared => ../shared
```

### Prerequisites

- Go 1.23+
- Docker (for server development with PostgreSQL)
- A code editor

### Quick start

```bash
# Build the CLI
cd app/cli && go build -o plandex .

# Build the server
cd app/server && go build -o plandex-server .

# Run tests
./scripts/test_modules.sh
```

### Server development

The server requires PostgreSQL and LiteLLM. The easiest way:

```bash
cd app
docker compose up -d plandex-postgres   # start postgres
cd server
DATABASE_URL="postgres://plandex:plandex@localhost:5432/plandex?sslmode=disable" \
  GOENV=development \
  go run .
```

## Project structure

```
app/
├── cli/              # CLI application (Go + Cobra + Bubbletea)
│   ├── api/          # HTTP client to server API
│   ├── auth/         # Authentication (cloud, local, API keys)
│   ├── cmd/          # Cobra commands
│   ├── plan_exec/    # Plan build & execution engine
│   ├── stream_tui/   # TUI for streaming AI responses
│   ├── term/         # Terminal utilities
│   └── ui/           # UI components (tables, prompts, etc.)
├── server/           # API server (Go + Gorilla Mux)
│   ├── db/           # Database queries & models
│   ├── handlers/     # HTTP handlers
│   ├── hooks/        # Plan lifecycle hooks (build, stream, apply)
│   ├── migrations/   # PostgreSQL migrations
│   ├── model/        # LLM model interaction layer
│   ├── routes/       # Route registration
│   └── syntax/       # Tree-sitter syntax validation
└── shared/           # Shared types & utilities
    ├── data_models.go    # Core data structures (Plan, Branch, Context)
    ├── plan_result.go    # Plan execution results
    ├── stream.go         # Streaming protocol types
    └── ...
```

## Code conventions

- **Formatting**: `gofmt` or `goimports`. CI checks this.
- **Comments**: Public functions and types should have doc comments.
- **Error handling**: Return errors up; log only at the top level.
- **Tests**: Use table-driven tests. Name test files `*_test.go`.

## Making changes

1. Fork the repo and create a branch.
2. Make your changes.
3. Run `go vet ./...` and `gofmt -l .` in the affected module.
4. Add tests if applicable.
5. Run `./scripts/test_modules.sh` to verify nothing breaks (it uses `-race` on supported platforms and automatically falls back on unsupported ones such as `android/arm64`).
6. Open a PR against `main`.

CI will automatically:
- Lint (`go vet`, `gofmt`, `staticcheck`)
- Test with platform-aware race handling (`scripts/test_modules.sh`)
- Validate the explicit non-race fallback path (`PLDX_FORCE_NO_RACE=1`)
- Build all modules
- Run an integration smoke test

## Where to help

- **Tests**: Coverage is sparse — adding tests is always welcome.
- **Documentation**: Improve docs, add examples.
- **CLI UX**: Bubbletea TUI improvements, keyboard shortcuts, color themes.
- **Server**: Model provider integrations, performance optimizations.
- **Syntax validation**: Expand tree-sitter language support.
- **Bug fixes**: Check the [issues](https://github.com/plandex-ai/plandex/issues) tab.

## Questions?

- [Discord](https://discord.gg/plandex-ai)
- [Discussions](https://github.com/plandex-ai/plandex/discussions)
