# Detailed Command Usage

This section provides detailed explanations and examples for each Plandex CLI command.

## General Commands

### `plandex new`

Start a new plan.

```bash
plandex new
```

You can also name your plan:

```bash
plandex new -n my-plan
```

### `plandex load`

Load files, directories, URLs, notes, or piped data into context.

```bash
plandex load file1.ts file2.ts
plandex load src -r  # Load a directory recursively
plandex load https://example.com/docs
plandex load -n "This is a note."
```

### `plandex tell`

Describe a task, ask a question, or chat.

```bash
plandex tell "Add a new feature to the project."
plandex tell -f prompt.txt  # Load a prompt from a file
```

### `plandex changes`

Review pending changes in a TUI.

```bash
plandex changes
```

### `plandex diff`

Review pending changes in 'git diff' format.

```bash
plandex diff
```

### `plandex apply`

Apply pending changes to project files.

```bash
plandex apply
```

### `plandex reject`

Reject pending changes to one or more project files.

```bash
plandex reject file1.ts
plandex reject --all  # Reject all pending changes
```

## Plan Management

### `plandex plans`

List plans.

```bash
plandex plans
```

### `plandex cd`

Set current plan by name or index.

```bash
plandex cd my-plan
plandex cd 1  # Set current plan by index
```

### `plandex current`

Show current plan.

```bash
plandex current
```

### `plandex delete-plan`

Delete a plan by name or index.

```bash
plandex delete-plan my-plan
plandex delete-plan 1  # Delete plan by index
```

### `plandex rename`

Rename the current plan.

```bash
plandex rename new-name
```

### `plandex archive`

Archive a plan.

```bash
plandex archive my-plan
```

### `plandex unarchive`

Unarchive a plan.

```bash
plandex unarchive my-plan
```

## Context Management

### `plandex ls`

List everything in context.

```bash
plandex ls
```

### `plandex rm`

Remove context by index, range, name, or glob.

```bash
plandex rm file1.ts
plandex rm 1  # Remove by index
plandex rm 1-3  # Remove a range of indices
```

### `plandex update`

Update outdated context.

```bash
plandex update
```

### `plandex clear`

Remove all context.

```bash
plandex clear
```

## Branch Management

### `plandex branches`

List plan branches.

```bash
plandex branches
```

### `plandex checkout`

Checkout or create a branch.

```bash
plandex checkout new-branch
plandex checkout existing-branch
```

### `plandex delete-branch`

Delete a branch by name or index.

```bash
plandex delete-branch branch-name
plandex delete-branch 1  # Delete branch by index
```

## History and Rewind

### `plandex log`

Show plan history.

```bash
plandex log
```

### `plandex rewind`

Rewind to a previous state.

```bash
plandex rewind 3  # Rewind 3 steps
plandex rewind a7c8d66  # Rewind to a specific commit
```

### `plandex convo`

Show plan conversation.

```bash
plandex convo
```

## Background Tasks

### `plandex ps`

List active and recently finished plan streams.

```bash
plandex ps
```

### `plandex connect`

Connect to an active plan stream.

```bash
plandex connect
```

### `plandex stop`

Stop an active plan stream.

```bash
plandex stop
```

## Model Management

### `plandex models`

Show plan model settings.

```bash
plandex models
```

### `plandex models default`

Show org-wide default model settings for new plans.

```bash
plandex models default
```

### `plandex models available`

Show all available models.

```bash
plandex models available
```

### `plandex set-model`

Update current plan model settings.

```bash
plandex set-model planner openai/gpt-4
```

### `plandex set-model default`

Update org-wide default model settings for new plans.

```bash
plandex set-model default planner openai/gpt-4
```

### `plandex models add`

Add a custom model.

```bash
plandex models add
```

### `plandex models delete`

Delete a custom model.

```bash
plandex models delete model-name
```

### `plandex model-packs`

Show all available model packs.

```bash
plandex model-packs
```

### `plandex model-packs create`

Create a new custom model pack.

```bash
plandex model-packs create
```

### `plandex model-packs delete`

Delete a custom model pack.

```bash
plandex model-packs delete model-pack-name
```

## Account Management

### `plandex sign-in`

Sign in, accept an invite, or create an account.

```bash
plandex sign-in
```

### `plandex invite`

Invite a user to join your org.

```bash
plandex invite user@example.com
```

### `plandex revoke`

Revoke an invite or remove a user from your org.

```bash
plandex revoke user@example.com
```

### `plandex users`

List users and pending invites in your org.

```bash
plandex users
```
