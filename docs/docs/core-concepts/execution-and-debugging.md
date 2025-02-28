
---
sidebar_position: 6
sidebar_label: Execution and Debugging
---

# Execution and Debugging

Plandex v2 includes powerful execution and automated debugging capabilities that help you run commands and automatically fix issues when they arise.

## Command Execution

After applying changes to your project files, you often need to run commands to verify that the changes work as expected. Plandex can execute these commands for you and capture their output.

### Execution Permissions

By default, Plandex can execute commands after changes are applied. This behavior is controlled by the `can-exec` configuration setting:

```bash
plandex set-config can-exec true  # Allow command execution (default)
plandex set-config can-exec false # Disable command execution
```

If you're concerned about security, you can disable command execution entirely. When disabled, Plandex will not execute any commands, even if they're part of a plan or explicitly requested.

### Automatic Execution

Plandex can automatically execute commands after changes are applied. This behavior is controlled by the `auto-exec` configuration setting:

```bash
plandex set-config auto-exec true  # Automatically execute commands
plandex set-config auto-exec false # Prompt before executing commands (default)
```

You can also override this setting for a specific command:

```bash
plandex apply --auto-exec  # Automatically execute commands after applying changes
plandex apply --no-exec    # Don't execute commands after applying changes
```

When using `plandex tell`, `plandex continue`, or `plandex build` with the `--apply` flag, you can also control command execution:

```bash
plandex tell "add a new route" --apply --auto-exec
plandex tell "add a new route" --apply --no-exec
```

## Debugging Commands

Plandex v2 includes a powerful `plandex debug` command that can repeatedly run any terminal command, continually making fixes based on the command's output until it succeeds.

### Using `plandex debug`

To use `plandex debug`, simply run it with the command you want to debug:

```bash
plandex debug 'npm test'
```

This will:

1. Run the specified command and check whether it succeeds or fails
2. If it fails, send the exit code and command output to the LLM
3. Apply the suggested fixes to your project files
4. Repeat until the command succeeds or the maximum number of tries is reached

### Number of Tries

By default, `plandex debug` will run the command up to 5 times before giving up. You can change this by providing a different number of tries as the first argument:

```bash
plandex debug 10 'npm test'  # Try up to 10 times
```

You can also configure the default number of tries with the `auto-debug-tries` configuration setting:

```bash
plandex set-config auto-debug-tries 10  # Set default to 10 tries
```

### Git Integration

Like the `apply` command, `plandex debug` can automatically commit changes to git after each successful fix:

```bash
plandex debug --commit 'npm test'      # Commit changes after each successful fix
plandex debug --skip-commit 'npm test' # Don't commit changes (default)
```

This behavior is also controlled by the `auto-commit` configuration setting:

```bash
plandex set-config auto-commit true  # Automatically commit changes
plandex set-config auto-commit false # Don't automatically commit changes (default)
```

### Commands That Succeed

If a command succeeds on the first try, `plandex debug` will exit immediately without making any model calls. This means you can use it for commands that may or may not succeed on the first try without worrying about unnecessary model usage.

```bash
plandex debug "echo 'ok'"  # Succeeds and immediately exits
```

### Automatic Debugging

Plandex can automatically debug failing commands after changes are applied. This behavior is controlled by the `auto-debug` configuration setting:

```bash
plandex set-config auto-debug true  # Automatically debug failing commands
plandex set-config auto-debug false # Don't automatically debug failing commands (default)
```

You can also override this setting for a specific command:

```bash
plandex apply --debug     # Automatically debug failing commands after applying changes
plandex apply --debug 10  # Automatically debug failing commands with 10 tries
```

When using `plandex tell`, `plandex continue`, or `plandex build` with the `--apply` flag, you can also control automatic debugging:

```bash
plandex tell "add a new route" --apply --debug
plandex tell "add a new route" --apply --debug 10
```

## Common Debugging Workflows

### Fixing Failing Tests

One of the most common use cases for `plandex debug` is fixing failing tests:

```bash
plandex debug 'npm test'
plandex debug 'go test ./...'
plandex debug 'pytest'
```

### Fixing Build Errors

Another common use case is fixing build errors:

```bash
plandex debug 'npm run build'
plandex debug 'go build'
plandex debug 'cargo build'
```

### Fixing Linting Errors

You can also use `plandex debug` to fix linting errors:

```bash
plandex debug 'npm run lint'
plandex debug 'golangci-lint run'
plandex debug 'cargo clippy'
```

### Fixing Type Errors

For TypeScript or other statically typed languages, you can use `plandex debug` to fix type errors:

```bash
plandex debug 'npm run typecheck'
plandex debug 'tsc --noEmit'
```

## Alternative Approaches

### Piping Into `plandex tell`

For a less automated approach that can accomplish the same thing, you can run your command and then pipe its output into `plandex tell`:

```bash
npm test | plandex tell 'npm test output'
```

This will work similarly to `plandex debug`, but without the automatic retries and changes. You can review the changes and then run `plandex apply` if you're happy with them.

