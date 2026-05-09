# Plandex Architecture

Plandex is a terminal-based AI coding agent capable of planning and executing complex, multi-file tasks. This document describes the internal architecture.

## High-Level Overview

```
┌─────────────────┐     HTTP/SSE      ┌──────────────────┐
│   plandex-cli   │ ◄──────────────► │  plandex-server   │
│   (Go + Cobra   │                   │  (Go + Gorilla    │
│    + Bubbletea) │                   │   Mux + sqlx)     │
└────────┬────────┘                   └────────┬─────────┘
         │                                     │
         │  Local filesystem                   │  PostgreSQL
         ▼                                     ▼
    Project dir                          ┌──────────┐
    (user code)                          │ Postgres │
                                         └────┬─────┘
                                              │
                                         ┌────┴─────┐
                                         │ LiteLLM  │
                                         │  Proxy   │
                                         └────┬─────┘
                                              │
                                    ┌─────────┼─────────┐
                                    ▼         ▼         ▼
                                 OpenAI   Anthropic   Google
```

The CLI communicates with the server over HTTP, with Server-Sent Events (SSE) for streaming responses. The server manages plans, branches, contexts, and LLM orchestration through a LiteLLM proxy sidecar.

## Repository Structure

```
app/
├── cli/              # plandex-cli — terminal client
│   ├── main.go       # Entry point, dependency injection
│   ├── cmd/          # Cobra commands (51 files)
│   ├── api/          # HTTP client to server API
│   ├── auth/         # Authentication (cloud, local, API keys)
│   ├── plan_exec/    # Plan build & execution orchestration
│   ├── stream_tui/   # Bubbletea TUI for streaming output
│   ├── stream/       # SSE stream protocol client
│   ├── format/       # Formatting utilities (time, filenames)
│   ├── fs/           # File system abstractions
│   ├── url/          # URL fetching and validation
│   ├── term/         # Terminal utilities
│   ├── ui/           # UI components (tables, prompts, etc.)
│   ├── lib/          # Core library (plan state, context checks)
│   ├── schema/       # JSON Schema validation
│   ├── types/        # CLI-specific types
│   ├── utils/        # General utilities
│   ├── version/      # Version info
│   └── upgrade.go    # Self-upgrade mechanism
├── server/           # plandex-server — API & LLM backend
│   ├── main.go       # Entry point
│   ├── routes/       # Route registration
│   ├── handlers/     # HTTP handler implementations
│   ├── db/           # Database queries & data access
│   ├── model/        # LLM model interaction & orchestration
│   │   ├── plan/     # Plan-level operations (tell, build, stream)
│   │   └── parse/    # Response parsing (subtasks, files, etc.)
│   ├── hooks/        # Plan lifecycle hooks (build, stream, apply)
│   ├── migrations/   # PostgreSQL migration files
│   ├── syntax/       # Tree-sitter syntax validation
│   ├── diff/         # Git diff generation
│   ├── types/        # Server-specific types (SafeMap, Reply)
│   ├── utils/        # XML, whitespace utilities
│   ├── email/        # Email sending (verification, invites)
│   ├── notify/       # Desktop notifications
│   ├── setup/        # Server setup (DB init, IP detection)
│   ├── shutdown/     # Graceful shutdown
│   └── host/         # Host detection (cloud vs local)
├── shared/           # plandex-shared — cross-cutting types
│   ├── data_models.go        # Core: Plan, Branch, Context, Org
│   ├── plan_result.go        # PlanResult, Replacements
│   ├── stream.go             # Stream message types
│   ├── context.go            # Context management types
│   ├── convo_message.go      # Conversation message models
│   ├── ai_models_*.go        # AI model configs, packs, providers
│   ├── auth.go               # Auth types (User, ApiKey)
│   ├── tokens.go             # Token counting
│   ├── file_maps.go          # Project file maps (tree-sitter)
│   ├── syntax.go             # Syntax validation types
│   └── rbac.go               # Role-based access control
└── docker-compose.yml        # Local dev environment
```

## Data Flow: Tell → Build → Apply

The core workflow follows three phases:

### 1. Tell (Prompt → Plan)

```
User types: plandex tell "Build a login page"

CLI (cmd/tell.go)
  → api.Client.Tell(planId, branch, prompt)
    → POST /api/plans/{id}/tell
      → server/handlers/tell_handler.go
        → model/plan/tell.go     # sends prompt to LLM
          → LiteLLM proxy        # routes to configured model
        ← streaming SSE response
    ← CLI receives SSE stream
  → stream_tui                  # renders streaming output in Bubbletea
```

### 2. Build (Plan → Implementation Steps)

```
CLI (cmd/build.go)
  → api.Client.Build(planId, branch)
    → POST /api/plans/{id}/build
      → hooks/build.go
        → model/plan/build.go   # LLM generates implementation plan
          → parse/subtasks.go   # parses subtasks from response
      → DB stores subtasks
    ← response with subtask list
```

### 3. Apply (Execute each subtask)

```
CLI (plan_exec/)
  → For each subtask:
    → Tell LLM to implement specific changes
      → server/hooks/stream.go      # streams file changes
        → server/hooks/apply.go     # applies changes to files
          → server/syntax/           # validates syntax (tree-sitter)
          → server/diff/diff.go      # computes git diffs
    ← CLI receives file changes via SSE
  → User reviews diffs in sandbox
  → User approves → changes applied to project files
```

## Key Components

### CLI (plandex-cli)

- **Cobra commands**: 51 command files under `cmd/` covering all operations (`tell`, `build`, `apply`, `new`, `continue`, `connect`, `stop`, `diffs`, `log`, `plans`, `branches`, `load`, `rm`, `config`, etc.)
- **Bubbletea TUI**: The `stream_tui/` package provides an interactive terminal UI for streaming LLM responses, with syntax-highlighted code blocks and real-time progress.
- **REPL mode**: When run without arguments, Plandex enters an interactive REPL with fuzzy auto-complete (via `go-prompt`).
- **Self-upgrade**: `upgrade.go` checks for new releases and performs in-place binary updates.

### Server (plandex-server)

- **Gorilla Mux router**: REST API with health routes, proxyable routes, and API routes.
- **PostgreSQL via sqlx**: All persistent state (plans, branches, contexts, users, orgs, model configs) stored in PostgreSQL. Migrations managed by `golang-migrate`.
- **LiteLLM proxy**: A Python-based sidecar (`litellm_proxy.py`) that provides a unified OpenAI-compatible API for multiple model providers (OpenAI, Anthropic, Google, OpenRouter, Ollama).
- **Streaming**: SSE-based streaming from LLM responses through server to CLI.
- **Tree-sitter**: Syntax validation via `go-tree-sitter` for 30+ languages during file edits.

### Shared (plandex-shared)

- Defines all cross-cutting types used by both CLI and server.
- Model configuration system: model packs, provider configs, compatibility matrices, custom model support.
- Data models: Plan, Branch, Context, ConvoMessage, PlanResult, Replacement, etc.
- Both CLI and server reference shared via Go module `replace` directive.

## Deployment

### Local Development

```bash
cd app
docker compose up -d          # starts PostgreSQL + server
```

### Production

- **Server**: Docker image `plandexai/plandex-server` published on Docker Hub
- **CLI**: Single binary distributed via `curl -sL https://plandex.ai/install.sh | bash`
- **CI/CD**: GitHub Actions workflows for CI (build/test/lint/integration), CLI release builds, and Docker image publishing. Test execution uses a platform-aware runner (`scripts/test_modules.sh`) to handle unsupported race-detector targets like `android/arm64`.

## Dependencies

### CLI

| Dependency | Purpose |
|-----------|---------|
| `cobra` | CLI command framework |
| `bubbletea` + `bubbles` + `lipgloss` | TUI framework |
| `go-openai` | OpenAI API client |
| `chromedp` | Browser automation (debug mode) |
| `go-prompt` | REPL with fuzzy completion |
| `survey` | Interactive prompts |
| `glamour` + `glow` | Markdown rendering |
| `goquery` | HTML parsing for URL fetching |
| `tiktoken-go` | Token counting |

### Server

| Dependency | Purpose |
|-----------|---------|
| `gorilla/mux` | HTTP router |
| `sqlx` + `lib/pq` | PostgreSQL access |
| `golang-migrate` | Schema migrations |
| `go-tree-sitter` | Syntax validation |
| `go-openai` | LLM API client |
| `aws-sdk-go` | S3 file storage |
| `beeep` | Desktop notifications |

## Version

Current: **2.2.1** (both CLI and server)
