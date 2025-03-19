---
sidebar_position: 11
sidebar_label: Configuration
---

# Configuration

Plandex v2 provides a flexible configuration system that lets you customize its behavior based on the task you're working on and your preferences.

## Viewing Config

```bash
plandex config           # View current plan's configuration
plandex config default   # View default configuration for new plans
```

## Modifying Config

```bash
plandex set-config                       # Select from a list of options
plandex set-config auto-apply true       # Set a specific option
plandex set-config default auto-mode basic   # Set default for new plans
```

## Key Settings

### Autonomy Level

Autonomy settings control the overall level of automation Plandex will use. See the [Autonomy](./autonomy.md) page for more details and shortcuts for setting autonomy levels.

| Setting               | Description                                                      | Default |
| --------------------- | ---------------------------------------------------------------- | ------- |
| `auto-mode`           | Overall autonomy level (`none`, `basic`, `plus`, `semi`, `full`) | `semi` |

### Plan Control

| Setting               | Description                                                      | Default |
| --------------------- | ---------------------------------------------------------------- | ------- |
| `auto-continue`       | Continue plans until completion                                  | `true`  |
| `auto-build`          | Build changes into pending updates                               | `true`  |
| `auto-apply`          | Apply changes to project files                                   | `false` |

### Context Management

| Setting                 | Description                              | Default |
| ----------------------- | ---------------------------------------- | ------- |
| `auto-update-context` | Update context when files change           | `true`  |
| `auto-load-context`     | Load context using project map           | `true`  |
| `smart-context`         | Load only necessary files for each step  | `true`  |

### Execution

| Setting                 | Description                              | Default |
| ----------------------- | ---------------------------------------- | ------- |
| `can-exec`              | Allow command execution (safety setting) | `true`  |
| `auto-exec`             | Automatically execute commands           | `true` |
| `auto-debug`            | Automatically debug commands             | `false` |
| `auto-debug-tries`      | Number of tries for automatic debugging  | `5`     |

### Version Control

| Setting                 | Description                              | Default |
| ----------------------- | ---------------------------------------- | ------- |
| `auto-commit`           | Commit changes to git when applied       | `true` |
| `auto-revert-on-rewind` | Revert project files when rewinding      | `true`  |


## Command Line Overrides

Settings can be overridden with command line flags:

```bash
# this will apply changes, automatically execute commands, and automatically debug regardless of your autonomy level and config settings
plandex tell "add a feature" --apply --auto-exec --debug
```

These overrides apply only to the current command execution and don't change the saved configuration.

See the [CLI Reference](../cli-reference.md) for a full list of command line flags for each command.

## REPL Commands

```
\config                # View current plan config
\config default        # View default config
\set-config            # Modify current plan config
\set-config default    # Modify default config for new plans
\set-auto              # Set autonomy level
\set-auto default      # Set default autonomy level for new plans
```
