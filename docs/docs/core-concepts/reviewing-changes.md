---
sidebar_position: 5
sidebar_label: Pending Changes
---

# Pending Changes

When you give Plandex a task, by default the changes aren't applied directly to your project files. Instead, they are accumulated in Plandex's version-controlled **sandbox** so that you can review them first.

## Review Menu

Once Plandex has finished with a task, you'll see a review menu with several hotkey options. These hotkeys act as shortcuts for the commands described below.

## Viewing Changes

### `plandex diff` / `plandex diff --ui`

When Plandex has finished with your task, you can review the proposed changes with the `plandex diff` command, which shows them in `git diff` format:

```bash
plandex diff
```

`--plain/-p`: Outputs the diff in plain text with no ANSI codes.

You can also view the changes in a local browser UI with the `plandex diff --ui` command:

```bash
plandex diff --ui
```

The UI view offers additional options:

- `--side-by-side/-s`: Show diffs in side-by-side view
- `--line-by-line/-l`: Show diffs in line-by-line view (default)

## Rejecting Files

If the plan's changes were applied incorrectly to a file, or you don't want to apply them for another reason, you can either [apply the changes](#applying-changes) and then fix the problems manually, _or_ you can reject the updates to that file and then make the proposed changes yourself manually.

To reject changes to a file (or multiple files), you can run `plandex reject`. You'll be prompted to select which files to reject.

```bash
plandex reject # select files to reject
```

You can reject _all_ currently pending files by passing the `--all` flag to the reject command, or you can pass a list of specific files to reject:

```bash
plandex reject --all
plandex reject file1.ts file2.ts
```

If you rejected a file due to the changes being applied incorrectly, but you still want to use the code, either scroll up and copy the changes from the plan's output or run `plandex convo` to output the full conversation and copy from there. Then apply the updates to that file yourself.

## Applying Changes

Once you're happy with the plan's changes, you can apply them to your project files with `plandex apply`:

```bash
plandex apply
```

### Apply Flags

Plandex v2 introduces several new flags for the `apply` command that give you more control over what happens after changes are applied:

- `--auto-update-context`: Automatically update context if files have changed (defaults to config value `auto-update-context`)
- `--no-exec`: Don't execute commands after successful apply (defaults to opposite of config value `can-exec`)
- `--auto-exec`: Automatically execute commands after successful apply without confirmation (defaults to config value `auto-exec`)
- `--debug`: Automatically execute and debug failing commands (optionally specify number of tries—default is 5, based on config values `auto-debug` and `auto-debug-tries`)
- `--commit/-c`: Commit changes to git after applying (defaults to config value `auto-commit`)
- `--skip-commit`: Don't commit changes to git (defaults to opposite of config value `auto-commit`)

### Git Integration

If you're in a git repository, Plandex will give you the option of grouping the changes into a git commit with an automatically generated commit message. Any uncommitted changes that were present in your working directory beforehand will be unaffected.

You can control git integration with these flags:

```bash
plandex apply --commit      # Commit changes to git after applying
plandex apply --skip-commit # Don't commit changes to git
```

Or you can set the `auto-commit` config value to `true` or `false` to control this behavior globally:

```bash
plandex set-config auto-commit true
plandex set-config default skip-commit true # Set the default value for new plans
```

### Command Execution

After applying changes, Plandex can automatically execute commands that were part of the plan. This is useful for running tests, starting servers, or performing other actions that verify the changes work as expected.

```bash
plandex apply --auto-exec   # Automatically execute commands after successful apply
plandex apply --no-exec     # Don't execute commands after successful apply
```

### Automatic Debugging

If a command fails after applying changes, Plandex can automatically attempt to debug and fix the issue:

```bash
plandex apply --debug       # Automatically debug failing commands (default 5 tries)
plandex apply --debug 10    # Automatically debug failing commands with 10 tries
```

## Auto-Applying Changes

If you want to skip the review step and automatically apply the changes from a plan immediately after it's complete, you can pass the `--apply/-a` flag to `plandex tell`, `plandex continue`, or `plandex build`:

```bash
plandex tell "add a new route" --apply
plandex continue --apply
plandex build --apply
```

You can combine this with other flags:

```bash
plandex tell "add a new route" --apply --commit      # Apply and commit changes
plandex tell "add a new route" --apply --auto-exec   # Apply and execute commands
plandex tell "add a new route" --apply --debug       # Apply and debug failing commands
```

## Configuration Settings

Plandex v2 introduces several configuration settings that control how changes are applied. You can view these settings with `plandex config`:

```bash
plandex config
```

The following settings affect how changes are applied:

- `auto-apply`: Whether changes are automatically applied after a plan is complete
- `auto-exec`: Whether commands are automatically executed after changes are applied
- `auto-debug`: Whether failing commands are automatically debugged
- `auto-debug-tries`: Number of tries for automatic debugging (default: 5)
- `auto-commit`: Whether changes are automatically committed to git after applying
- `can-exec`: Whether commands can be executed at all (safety setting)

You can modify these settings with `plandex set-config`:

```bash
plandex set-config auto-apply true
plandex set-config auto-exec true
plandex set-config auto-debug true
plandex set-config auto-debug-tries 10
plandex set-config auto-commit true
plandex set-config can-exec true
```

## Interaction with Autonomy Levels

The behavior of the `apply` command is also affected by the [autonomy level](./autonomy.md) you've set for your plan:

| Setting       | None | Basic | Plus | Semi | Full |
| ------------- | ---- | ----- | ---- | ---- | ---- |
| `auto-apply`  | ❌   | ❌    | ❌   | ❌   | ✅   |
| `auto-exec`   | ❌   | ❌    | ❌   | ❌   | ✅   |
| `auto-debug`  | ❌   | ❌    | ❌   | ❌   | ✅   |
| `auto-commit` | ❌   | ❌    | ✅   | ✅   | ✅   |

If you're using the `full` autonomy level, changes will be automatically applied to your project files, commands will be automatically executed, and failing commands will be automatically debugged.

For other autonomy levels, you'll need to manually apply changes, but you can still use the flags described above to override the default behavior for a specific command.

## Recommended Workflow

For most users, we recommend the following workflow:

1. Use `plandex tell` to give Plandex a task
2. Review the proposed changes with `plandex diff` or `plandex diff --ui`
3. Apply the changes with `plandex apply`
4. If commands need to be executed, decide whether to execute them automatically or manually

For users who want more automation, we recommend:

1. Set the autonomy level to `full` with `plandex set-auto full`
2. Use `plandex tell` to give Plandex a task
3. Let Plandex automatically apply changes, execute commands, and debug failing commands

For users who want more control, we recommend:

1. Set the autonomy level to `plus` or `semi` with `plandex set-auto plus` or `plandex set-auto semi`
2. Use `plandex tell` to give Plandex a task
3. Review the proposed changes with `plandex diff` or `plandex diff --ui`
4. Apply the changes with `plandex apply`
5. Manually execute any commands that need to be run
   If you want to skip the review step and automatically apply the changes from a plan immediately after it's complete, you can pass the `--apply/-a` flag to `plandex tell`, `plandex continue`, or `plandex build`.

If you do this, you can also pass the `--commit/-c` flag to commit the automatically applied changes to git.
