---
sidebar_position: 6
sidebar_label: CLI Reference
---

# CLI Reference

All Plandex CLI commands and their options.

## Usage

```bash
plandex [command] [flags]
pdx [command] [flags] # 'pdx' is an alias for 'plandex'
```

## Help

Built-in help.

```bash
plandex help
pdx h # alias
```

`--all/-a`: List all commands.

For help on a specific command, use:

```bash
plandex [command] --help
```

## Plans

### new

Start a new plan.

```bash
plandex new
plandex new -n new-plan # with name
```

`--name/-n`: Name of the new plan. The name is generated automatically after first prompt if no name is specified on creation.

### plans

List plans. Output includes index, when each plan was last updated, the current branch of each plan, the number of tokens in context, and the number of tokens in the conversation (prior to summarization).

Includes full details on plans in current directory. Also includes names of plans in parent directories and child directories.

```bash
plandex plans
plandex plans --archived # list archived plans only

pdx pl # alias
```

`--archived/-a`: List archived plans only.

### current

Show current plan. Output includes when the plan was last updated and created, the current branch, the number of tokens in context, and the number of tokens in the conversation (prior to summarization).

```bash
plandex current
pdx cu # alias
```

### cd

Set current plan by name or index.

```bash
plandex cd # select from a list of plans
plandex cd some-plan # by name
plandex cd 4 # by index in `plandex plans`
```

With no arguments, Plandex prompts you with a list of plans to select from.

With one argument, Plandex selects a plan by name or by index in the `plandex plans` list.

### delete-plan

Delete a plan by name or index.

```bash
plandex delete-plan # select from a list of plans
plandex delete-plan some-plan # by name
plandex delete-plan 4 # by index in `plandex plans`

pdx dp # alias
```

With no arguments, Plandex prompts you with a list of plans to select from.

With one argument, Plandex deletes a plan by name or by index in the `plandex plans` list.

### rename

Rename the current plan.

```bash
plandex rename # prompt for new name
plandex rename new-name # set new name
```

With no arguments, Plandex prompts you for a new name.

With one argument, Plandex sets the new name.

### archive

Archive a plan.

```bash
plandex archive # select from a list of plans
plandex archive some-plan # by name
plandex archive 4 # by index in `plandex plans`

pdx arc # alias
```

With no arguments, Plandex prompts you with a list of plans to select from.

With one argument, Plandex archives a plan by name or by index in the `plandex plans` list.

### unarchive

Unarchive a plan.

```bash
plandex unarchive # select from a list of archived plans
plandex unarchive some-plan # by name
plandex unarchive 4 # by index in `plandex plans --archived`
pdx unarc # alias
```

## Context

### load

Load files, directories, directory layouts, URLs, notes, images, or piped data into context.

```bash
plandex load component.ts # single file
plandex load component.ts action.ts reducer.ts # multiple files
plandex load lib -r # loads lib and all its subdirectories
plandex load tests/**/*.ts # loads all .ts files in tests and its subdirectories
plandex load . --tree # loads the layout of the current directory and its subdirectories (file names only)
plandex load https://redux.js.org/usage/writing-tests # loads the text-only content of the url
npm test | plandex load # loads the output of `npm test`
plandex load -n 'add logging statements to all the code you generate.' # load a note into context
plandex load ui-mockup.png # load an image into context

pdx l component.ts # alias
```

`--recursive/-r`: Load an entire directory and all its subdirectories.

`--tree`: Load directory tree layout with file names only.

`--note/-n`: Load a note into context.

`--force/-f`: Load files even when ignored by .gitignore or .plandexignore.

`--detail/-d`: Image detail level when loading an image (high or low)—default is high. See https://platform.openai.com/docs/guides/vision/low-or-high-fidelity-image-understanding for more info.

### ls

List everything in context. Output includes index, name, type, token size, when the context added, and when the context was last updated.

```bash
plandex ls
```

### rm

Remove context by index, range, name, or glob.

```bash
plandex rm

```

### update

Update outdated context.

```bash
plandex update
```

### clear

Remove all context.

```bash
plandex clear
```

## Control

### tell

Describe a task, ask a question, or chat.

```bash
plandex tell
```

### continue

Continue the plan.

```bash
plandex continue
```

### build

Build any pending changes.

```bash
plandex build
```

## Changes

### diff

Review pending changes in 'git diff' format.

```bash
plandex diff
```

### changes

Review pending changes in a TUI.

```bash
plandex changes
```

### apply

Apply pending changes to project files.

```bash
plandex apply
```

### reject

Reject pending changes to one or more project files.

```bash
plandex reject
```

## History

### log

Show plan history.

```bash
plandex log
```

### rewind

Rewind to a previous state.

```bash
plandex rewind
```

### convo

Show plan conversation.

```bash
plandex convo
```

## Branches

### branches

List plan branches.

```bash
plandex branches
```

### checkout

Checkout or create a branch.

```bash
plandex checkout
```

### delete-branch

Delete a branch by name or index.

```bash
plandex delete-branch
```

## Background Tasks / Streams

### ps

List active and recently finished plan streams.

```bash
plandex ps
```

### connect

Connect to an active plan stream.

```bash
plandex connect
```

### stop

Stop an active plan stream.

```bash
plandex stop
```


## Models

### models

Show plan model settings.

```bash
plandex models
```

### models default

Show org-wide default model settings for new plans.

```bash
plandex models default
```

### models available

Show all available models.

```bash
plandex models available
```

### set-model

Update current plan model settings.

```bash
plandex set-model
```

### set-model default

Update org-wide default model settings for new plans.

```bash
plandex set-model default
```

### models add

Add a custom model.

```bash
plandex models add
```

### models delete

Delete a custom model.

```bash
plandex models delete
```

### model-packs

Show all available model packs.

```bash
plandex model-packs
```

### model-packs create

Create a new custom model pack.

```bash
plandex model-packs create
```

### model-packs delete

Delete a custom model pack.

```bash
plandex model-packs delete
```


## Account Management

### sign-in

Sign in, accept an invite, or create an account.

```bash
plandex sign-in
```

### invite

Invite a user to join your org.

```bash
plandex invite
```

### revoke

Revoke an invite or remove a user from your org.

```bash
plandex revoke
```

### users

List users and pending invites in your org.

```bash
plandex users
```

