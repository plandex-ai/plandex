---
sidebar_position: 2
sidebar_label: Autonomy
---

# Autonomy Levels

Plandex v2 offers five different levels of autonomy. Each autonomy level controls:

- Automatic context loading and management
- Automatic plan continuation
- Automatic building of changes into pending updates
- Automatic application of changes to project files
- Automatic command execution and debugging
- Automatic git commits after changes are applied successfully

## The Five Autonomy Levels

### None

Complete manual control with no automation:

- Manual context loading
- Manual plan continuation
- Manual building of changes
- Manual application of changes
- Manual command execution

### Basic (equivalent to Plandex v1 autonomy level)

Minimal automation:

- Manual context loading
- Auto-continue plans until completion
- Manual building of changes
- Manual application of changes
- Manual command execution

### Plus

Smart context management:

- Manual initial context loading
- Auto-continue plans until completion
- Smart context management (only loads necessary files during implementation steps)
- Auto-update context when files change
- Auto-build changes into pending updates
- Manual application of changes
- Manual command execution
- Auto-commit changes to git when applied

### Semi (default level)

Automatic context loading:

- Auto-load context using project map
- Auto-continue plans until completion
- Smart context management
- Auto-update context when files change
- Auto-build changes into pending updates
- Manual application of changes
- Manual command execution
- Auto-commit changes to git when applied

### Full

Complete automation:

- Auto-load context using project map
- Auto-continue plans until completion
- Smart context management
- Auto-update context when files change
- Auto-build changes into pending updates
- Auto-apply changes to project files
- Auto-execute commands after successful apply
- Auto-debug failing commands
- Auto-commit changes to git when applied

## Autonomy Matrix

| Feature               | None | Basic | Plus | Semi | Full |
| --------------------- | ---- | ----- | ---- | ---- | ---- |
| `auto-continue`       | ❌   | ✅    | ✅   | ✅   | ✅   |
| `auto-build`          | ❌   | ❌    | ✅   | ✅   | ✅   |
| `auto-load-context`   | ❌   | ❌    | ❌   | ✅   | ✅   |
| `smart-context`       | ❌   | ❌    | ✅   | ✅   | ✅   |
| `auto-update-context` | ❌   | ❌    | ✅   | ✅   | ✅   |
| `auto-apply`          | ❌   | ❌    | ❌   | ❌   | ✅   |
| `auto-exec`           | ❌   | ❌    | ❌   | ❌   | ✅   |
| `auto-debug`          | ❌   | ❌    | ❌   | ❌   | ✅   |
| `auto-commit`         | ❌   | ❌    | ✅   | ✅   | ✅   |

## Setting Autonomy Levels

### Using the CLI

```bash
# For current plan
plandex set-auto none    # No automation
plandex set-auto basic   # Auto-continue only
plandex set-auto plus    # Smart context management
plandex set-auto semi    # Auto-load context
plandex set-auto full    # Full automation

# For default settings on new plans
plandex set-auto default basic   # Set default to basic
```

### When Starting the REPL

```bash
plandex --no-auto    # Start with 'None'
plandex --basic      # Start with 'Basic'
plandex --plus       # Start with 'Plus'
plandex --semi       # Start with 'Semi'
plandex --full       # Start with 'Full'
```

### When Creating a New Plan

```bash
plandex new --no-auto    # Create with 'None'
plandex new --basic      # Create with 'Basic'
plandex new --plus       # Create with 'Plus'
plandex new --semi       # Create with 'Semi'
plandex new --full       # Create with 'Full'
```

### Using the REPL

```
\set-auto none    # Set to None
\set-auto basic   # Set to Basic
\set-auto plus    # Set to Plus
\set-auto semi    # Set to Semi-Auto
\set-auto full    # Set to Full-Auto
```

### Custom Autonomy Config

For more details on configuration options, see the [Configuration](./configuration.md) page.

You can give a plan custom autonomy settings by setting config values directly:

```bash
plandex set-config auto-continue true
plandex set-config auto-build true
plandex set-config auto-load-context true
```

For more details on configuration options, see the [Configuration](./configuration.md) page.

## Safety

Be extremely careful with full auto mode! It can make many changes quickly without any prompting or review, and can run commands that could potentially be destructive to your system.

It's a good idea to make sure your git state is clean, and to check out an isolated branch before running these commands.
