---
sidebar_position: 10
sidebar_label: Configuration
---

# Configuration

Plandex v2 provides a flexible configuration system that lets you customize its behavior to match your workflow preferences.

## Viewing Configuration

```bash
plandex config           # View current plan's configuration
plandex config default   # View default configuration for new plans
```

## Modifying Configuration

```bash
plandex set-config                       # Select from a list of options
plandex set-config auto-apply true       # Set a specific option
plandex set-config default auto-mode basic   # Set default for new plans
```

## Key Configuration Settings

### Autonomy Settings

| Setting               | Description                                                      | Default |
| --------------------- | ---------------------------------------------------------------- | ------- |
| `auto-mode`           | Overall autonomy level (`none`, `basic`, `plus`, `semi`, `full`) | `basic` |
| `auto-continue`       | Continue plans until completion                                  | `true`  |
| `auto-build`          | Build changes into pending updates                               | `true`  |
| `auto-load-context`   | Load context using project map                                   | `false` |
| `smart-context`       | Load only necessary files for each step                          | `false` |
| `auto-update-context` | Update context when files change                                 | `false` |
| `auto-apply`          | Apply changes to project files                                   | `false` |
| `auto-exec`           | Execute commands after applying changes                          | `false` |
| `auto-debug`          | Debug failing commands                                           | `false` |
| `auto-commit`         | Commit changes to git when applied                               | `false` |

### Execution Settings

| Setting                 | Description                              | Default |
| ----------------------- | ---------------------------------------- | ------- |
| `can-exec`              | Allow command execution (safety setting) | `true`  |
| `auto-debug-tries`      | Number of tries for automatic debugging  | `5`     |
| `auto-revert-on-rewind` | Revert project files when rewinding      | `true`  |

## Setting Autonomy Levels

Instead of configuring individual settings, you can use predefined autonomy levels:

```bash
plandex set-auto none    # No automation, step-by-step
plandex set-auto basic   # Auto-continue plans only
plandex set-auto plus    # Smart context management
plandex set-auto semi    # Auto-load context
plandex set-auto full    # Full automation
```

For default settings on new plans:

```bash
plandex set-auto default semi   # Set default to semi-auto
```

## Autonomy Level Comparison

| Setting               | None | Basic | Plus | Semi | Full |
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

## Command Line Overrides

Many settings can be overridden with command line flags:

```bash
plandex tell "add a feature" --apply --auto-exec --debug
```

These overrides apply only to the current command execution and don't change the saved configuration.

## REPL Commands

```
\config                # View current plan config
\config default        # View default config
\set-config            # Modify current plan config
\set-auto              # Set autonomy level
```
