---
sidebar_position: 10
sidebar_label: Autonomy Levels
---

# Autonomy

Plandex v2 offers multiple levels of autonomy with pre-set config. Each autonomy level controls:

- Context loading and management
- Plan continuation through multiple steps
- Building of changes into pending updates
- Application of changes to project files
- Command execution and debugging
- Git commits after changes are applied successfully

## Autonomy Matrix

| Feature               | None | Basic | Plus | Semi | Full |
| --------------------- | ---- | ----- | ---- | ---- | ---- |
| `auto-continue`       | ❌   | ✅    | ✅   | ✅   | ✅   |
| `auto-build`          | ❌   | ✅    | ✅   | ✅   | ✅   |
| `auto-load-context`   | ❌   | ❌    | ❌   | ✅   | ✅   |
| `smart-context`       | ❌   | ❌    | ✅   | ✅   | ✅   |
| `auto-update-context` | ❌   | ❌    | ✅   | ✅   | ✅   |
| `auto-apply`          | ❌   | ❌    | ❌   | ❌   | ✅   |
| `can-exec`            | ❌   | ❌    | ✅   | ✅   | ✅   |
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

## Autonomy Levels

### None

Complete manual control with no automation:

- Manual context loading
- Context updates require approval
- Manual plan continuation
- Manual building of changes
- Manually apply changes
- Command execution disabled

### Basic

_Equivalent to Plandex v1 autonomy level_

Minimal automation:

- Manual context loading
- Context updates require approval
- Auto-continue plans until completion
- Auto-build changes into pending updates
- Manually apply changes
- Command execution disabled

### Plus

Smart context management and manual command execution:

- Manual context loading
- Auto-update context when files change
- Auto-continue plans until completion
- Smart context management (only loads necessary files during implementation steps)
- Auto-build changes into pending updates
- Manually apply changes
- Manual command execution
- Auto-commit changes to git when applied

### Semi

_Default autonomy level for a fresh Plandex v2 install_

Automatic context loading:

- Auto-load context using project map
- Auto-update context when files change
- Auto-continue plans until completion
- Smart context management
- Auto-build changes into pending updates
- Manually apply changes
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

### Custom

You can give a plan custom autonomy settings by setting config values directly:

```bash
plandex set-config auto-continue true
plandex set-config auto-build true
plandex set-config auto-load-context true
```

[More details on configuration](./configuration.md)

## Safety

Be extremely careful with full auto mode! It can make many changes quickly without any prompting or review, and can run commands that could potentially be destructive to your system.

It's a good idea to make sure your git state is clean, and to check out an isolated branch before running these commands.
