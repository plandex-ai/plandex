
---
sidebar_position: 6
sidebar_label: Execution and Debugging
---

# Execution and Debugging

Plandex v2 includes powerful execution and automated debugging capabilities to help you run commands and fix issues automatically.

## Command Execution

### Execution Permissions

Control whether Plandex can execute commands:

```bash
plandex set-config can-exec true  # Allow command execution (default)
plandex set-config can-exec false # Disable command execution
```

### Automatic Execution

Control whether commands are executed automatically after applying changes:

```bash
plandex set-config auto-exec true  # Auto-execute commands
plandex set-config auto-exec false # Prompt before executing (default)

# Override for specific commands
plandex apply --auto-exec  # Auto-execute after applying
plandex apply --no-exec    # Don't execute after applying

# With tell/continue/build
plandex tell "add a route" --apply --auto-exec
```

## Debugging Commands

### Using `plandex debug`

The `plandex debug` command repeatedly runs a terminal command, making fixes until it succeeds:

```bash
plandex debug 'npm test'  # Try up to 5 times (default)
plandex debug 10 'npm test'  # Try up to 10 times
```

This will:
1. Run the command and check for success/failure
2. If it fails, send the output to the LLM
3. Apply suggested fixes to your project files
4. Repeat until success or max tries reached

### Number of Tries

Configure the default number of tries:

```bash
plandex set-config auto-debug-tries 10  # Set default to 10 tries
```

### Git Integration

Control whether changes are committed during debugging:

```bash
plandex debug --commit 'npm test'      # Commit changes after fixes
plandex debug --skip-commit 'npm test' # Don't commit changes (default)

# Global setting
plandex set-config auto-commit true  # Auto-commit changes
```

### Automatic Debugging

Control whether failing commands are automatically debugged:

```bash
plandex set-config auto-debug true  # Auto-debug failing commands
plandex set-config auto-debug false # Don't auto-debug (default)

# Override for specific commands
plandex apply --debug     # Auto-debug failing commands
plandex apply --debug 10  # Auto-debug with 10 tries

# With tell/continue/build
plandex tell "add a route" --apply --debug
```

## Common Debugging Workflows

### Fixing Failing Tests

```bash
plandex debug 'npm test'
plandex debug 'go test ./...'
plandex debug 'pytest'
```

### Fixing Build Errors

```bash
plandex debug 'npm run build'
plandex debug 'go build'
plandex debug 'cargo build'
```

### Fixing Linting Errors

```bash
plandex debug 'npm run lint'
plandex debug 'golangci-lint run'
```

### Fixing Type Errors

```bash
plandex debug 'npm run typecheck'
plandex debug 'tsc --noEmit'
```

## Alternative Approaches

### Piping Into `plandex tell`

For a less automated approach:

```bash
npm test | plandex tell 'npm test output'
```

This works similarly to `plandex debug` but without automatic retries. You can review changes before applying them.

## Interaction with Autonomy Levels

Execution and debugging behavior is affected by your [autonomy level](./autonomy.md):

| Setting | None | Basic | Plus | Semi | Full |
| ------- | ---- | ----- | ---- | ---- | ---- |
| `auto-exec` | ❌ | ❌ | ❌ | ❌ | ✅ |
| `auto-debug` | ❌ | ❌ | ❌ | ❌ | ✅ |

With `full` autonomy, commands are automatically executed and debugged. For other levels, you need to manually execute and debug commands (or override with flags).

## Configuration Settings

Key settings that control execution and debugging:

```bash
plandex set-config can-exec true         # Allow command execution
plandex set-config auto-exec true        # Auto-execute commands
plandex set-config auto-debug true       # Auto-debug failing commands
plandex set-config auto-debug-tries 10   # Set debug tries
```

## Security Considerations

- Plandex only executes commands that are part of a plan or explicitly requested
- Plandex won't execute commands with elevated privileges unless explicitly requested
- You can disable command execution with `plandex set-config can-exec false`
- Always review commands before allowing them to run
- Be careful with `auto-exec` and `auto-debug` as they execute without prompting

## Recommended Workflow

### Standard Workflow
1. Use `plandex tell` to give Plandex a task
2. Review changes with `plandex diff`
3. Apply changes with `plandex apply`
4. If a command fails, use `plandex debug` to fix it

### Automated Workflow
1. Set autonomy to `full` with `plandex set-auto full`
2. Use `plandex tell` to give Plandex a task
3. Let Plandex automatically apply, execute, and debug

### Controlled Workflow
1. Set autonomy to `plus` or `semi`
2. Use `plandex tell` to give Plandex a task
3. Review changes with `plandex diff`
4. Apply changes with `plandex apply`
5. Manually execute commands
6. If a command fails, use `plandex debug` to fix it
```

This will work similarly to `plandex debug`, but without the automatic retries and changes. You can review the changes and then run `plandex apply` if you're happy with them.

