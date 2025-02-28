---
sidebar_position: 6
sidebar_label: Execution and Debugging
---

# Execution and Debugging

Plandex v2 includes powerful execution control and automated debugging capabilities.

## Command Execution

### Execution Config

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
3. Tentatively apply suggested fixes to your project files
4. If command is succesful after fixes, commit changes (if auto-commit is enabled). Otherwise, roll back changes and return to step 2.
5. Repeat until success or max tries reached

### Number of Tries

Configure the default number of tries:

```bash
plandex set-config auto-debug-tries 10  # Set default to 10 tries
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

| Setting      | None | Basic | Plus | Semi | Full |
| ------------ | ---- | ----- | ---- | ---- | ---- |
| `auto-exec`  | ❌   | ❌    | ❌   | ❌   | ✅   |
| `auto-debug` | ❌   | ❌    | ❌   | ❌   | ✅   |

With `full` autonomy, commands are automatically executed and debugged after changes are applied. For other levels, you'll be prompted to approve execution and debugging.

## Configuration Settings

Key settings that control execution and debugging:

```bash
plandex set-config can-exec true         # Allow command execution
plandex set-config auto-exec true        # Auto-execute commands
plandex set-config auto-debug true       # Auto-debug failing commands
plandex set-config auto-debug-tries 10   # Set debug tries
```

## Safety

Needless to say, you should be extremely careful when using `auto-exec`, `auto-debug`, and the `debug` command. They can make many changes quickly without any prompting or review, and can run commands that could potentially be destructive to your system. While the best LLMs are quite trustworthy when it comes to running commands and are unlikely to cause harm, it still pays to be cautious.

It's a good idea to make sure your git state is clean, and to check out an isolated branch before running these commands.
