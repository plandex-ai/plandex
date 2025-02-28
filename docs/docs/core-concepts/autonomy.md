
---
sidebar_position: 2
sidebar_label: Autonomy
---

# Autonomy Levels

Plandex v2 offers five autonomy levels that control how much automation is applied during your workflow.

## Overview

Each autonomy level controls:
- Context loading and management
- Plan continuation
- Building changes into pending updates
- Applying changes to project files
- Command execution and debugging
- Git integration

## The Five Autonomy Levels

### None

Complete manual control with no automation:
- Manual context loading
- Manual plan continuation
- Manual building of changes
- Manual application of changes
- Manual command execution

### Basic

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
- Smart context management (only loads necessary files)
- Auto-update context when files change
- Auto-build changes into pending updates
- Manual application of changes
- Manual command execution
- Auto-commit changes to git when applied

### Semi

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

## Feature Comparison

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

### Using Configuration

You can also set individual configuration options:

```bash
plandex set-config auto-continue true
plandex set-config auto-build true
plandex set-config auto-load-context true
# etc.
```

For more details on configuration options, see the [Configuration](./configuration.md) page.

## Recommended Workflow

- New users: Start with basic or plus to learn how Plandex works
- Experienced users: Use semi or full for maximum efficiency
- Critical production tasks: Use lower autonomy levels for careful review
- Exploratory tasks: Use full for quick iteration and experimentation
    
