---
sidebar_position: 8
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

List everything in the current plan's context. Output includes index, name, type, token size, when the context added, and when the context was last updated.

```bash
plandex ls

plandex list-context # longer alias
```

### rm

Remove context by index, range, name, or glob.

```bash
plandex rm some-file.ts # by name
plandex rm app/**/*.ts # by glob pattern
plandex rm 4 # by index in `plandex ls`
plandx rm 2-4 # by range of indices

plandex remove # longer alias
plandex unload # longer alias
```

### update

Update any outdated context.

```bash
plandex update
pdx u # alias
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
plandex tell -f prompt.txt # from file
plandex tell # open vim to write prompt
plandex tell "add a cancel button to the left of the submit button" # inline

pdx t # alias
```

`--file/-f`: File path containing prompt.

`--stop/-s`: Stop after a single model response (don't auto-continue).

`--no-build/-n`: Don't build proposed changes into pending file updates.

`--bg`: Run task in the background.

### continue

Continue the plan.

```bash
plandex continue

pdx c # alias
```

`--stop/-s`: Stop after a single model response (don't auto-continue).

`--no-build/-n`: Don't build proposed changes into pending file updates.

`--bg`: Run task in the background.

### build

Build any unbuilt pending changes from the plan conversation.

```bash
plandex build
pdx b # alias
```

`--bg`: Build in the background.

## Changes

### diff

Review pending changes in 'git diff' format.

```bash
plandex diff
```

`--plain/-p`: Output diffs in plain text with no ANSI codes.

### changes

Review pending changes in a TUI.

```bash
plandex changes
```

### apply

Apply pending changes to project files.

```bash
plandex apply
pdx ap # alias
```

`--yes/-y`: Skip confirmation.

### reject

Reject pending changes to one or more project files.

```bash
plandex reject file.ts # one file
plandex reject file.ts another-file.ts # multiple files
plandex reject --all # all pending files

pdx rj file.ts # alias
```

`--all/-a`: Reject all pending files.

## History

### log

Show plan history.

```bash
plandex log

plandex history # alias
plandex logs # alias
```

### rewind

Rewind to a previous state.

```bash
plandex rewind # rewind 1 step
plandex rewind 3 # rewind 3 steps
plandex rewind a7c8d66 # rewind to a specific step from `plandex log`
```

With no arguments, Plandex rewinds one step.

With one argument, Plandex rewinds the specified number of steps (if an integer is passed) or rewinds to the specified step (if a hash from `plandex log` is passed).

### convo

Show the current plan's conversation.

```bash
plandex convo
plandex convo 1 # show a specific message
plandex convo 1-5 # show a range of messages
plandex convo 3- # show all messages from 3 to the end
```

`--plain/-p`: Output conversation in plain text with no ANSI codes.

### summary

Show the latest summary of the current plan.

```bash
plandex summary
```

`--plain/-p`: Output summary in plain text with no ANSI codes.

## Branches

### branches

List plan branches. Output includes index, name, when the branch was last updated, the number of tokens in context, and the number of tokens in the conversation (prior to summarization).

```bash
plandex branches
pdx br # alias
```

### checkout

Checkout or create a branch.

```bash
plandex checkout # select from a list of branches or prompt to create a new branch
plandex checkout some-branch # checkout by name or create a new branch with that name

pdx co # alias
```

### delete-branch

Delete a branch by name or index.

```bash
plandex delete-branch # select from a list of branches
plandex delete-branch some-branch # by name
plandex delete-branch 4 # by index in `plandex branches`

pdx db # alias
```

With no arguments, Plandex prompts you with a list of branches to select from.

With one argument, Plandex deletes a branch by name or by index in the `plandex branches` list.

## Background Tasks / Streams

### ps

List active and recently finished plan streams. Output includes stream ID, plan name, branch name, when the stream was started, and the stream's status (active, finished, stopped, errored, or waiting for a missing file to be selected).

```bash
plandex ps
```

### connect

Connect to an active plan stream.

```bash
plandex connect # select from a list of active streams
plandex connect a4de # by stream ID in `plandex ps`
plandex connect some-plan main # by plan name and branch name
```

With no arguments, Plandex prompts you with a list of active streams to select from.

With one argument, Plandex connects to a stream by stream ID in the `plandex ps` list.

With two arguments, Plandex connects to a stream by plan name and branch name.

### stop

Stop an active plan stream.

```bash
plandex stop # select from a list of active streams
plandex stop a4de # by stream ID in `plandex ps`
plandex stop some-plan main # by plan name and branch name
```

With no arguments, Plandex prompts you with a list of active streams to select from.

With one argument, Plandex connects to a stream by stream ID in the `plandex ps` list.

With two arguments, Plandex connects to a stream by plan name and branch name.

## Models

### models

Show current plan models and model settings.

```bash
plandex models
```

### models default

Show org-wide default models and model settings for new plans.

```bash
plandex models default
```

### models available

Show available models.

```bash
plandex models available # show all available models
plandex models available --custom # show available custom models only
```

`--custom`: Show available custom models only.

### set-model

Update current plan models or model settings.

```bash
plandex set-model # select from a list of models and settings
plandex set-model planner openai/gpt-4 # set the model for a role
plandex set-model gpt-4-turbo-latest # set the current plan's model pack by name (sets all model roles at once—see `model-packs` below)
plandex set-model builder temperature 0.1 # set a model setting for a role
plandex set-model max-tokens 4000 # set the planner model overall token limit to 4000
plandex set-model max-convo-tokens 20000  # set how large the conversation can grow before Plandex starts using summaries
```

With no arguments, Plandex prompts you to select from a list of models and settings.

With arguments, can take one of the following forms:

- `plandex set-model [role] [model]`: Set the model for a role.
- `plandex set-model [model-pack]`: Set the current plan's model pack by name.
- `plandex set-model [role] [setting] [value]`: Set a model setting for a role.
- `plandex set-model [setting] [value]`: Set a model setting for the current plan.

Models are specified as `provider/model-name`, e.g. `openai/gpt-4`, `openrouter/anthropic/claude-opus-3`, `together/mistralai/Mixtral-8x22B-Instruct-v0.1`, etc.

See all the model roles [here](./models/roles.md).

Model role settings:

- `temperature`: Higher temperature means more randomness, which can produce more creativity but also more errors.
- `top-p`: Top-p sampling is a way to prevent the model from generating improbable text by only considering the most likely tokens.

Plan settings:

- `max-tokens`: The overall token limit for the planner model.
- `max-convo-tokens`: How large the conversation can grow before Plandex starts using summaries.
- `reserved-output-tokens`: The number of tokens reserved for output from the model.

### set-model default

Update org-wide default model settings for new plans.

```bash
plandex set-model default # select from a list of models and settings
plandex set-model default planner openai/gpt-4 # set the model for a role
# etc. — same options as `set-model` above
```

Works exactly the same as `set-model` above, but sets the default model settings for all new plans instead of only the current plan.

### models add

Add a custom model.

```bash
plandex models add
```

Plandex will prompt you for all required information to add a custom model.

### models delete

Delete a custom model.

```bash
plandex models delete # select from a list of custom models
plandex models delete some-model # by name
plandex models delete 4 # by index in `plandex models available --custom`
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

Plandex will prompt you for all required information to create a custom model pack.

### model-packs delete

Delete a custom model pack.

```bash
plandex model-packs delete
plandex model-packs delete some-model-pack # by name
plandex model-packs delete 4 # by index in `plandex model-packs --custom`
```

## Account Management

### sign-in

Sign in, accept an invite, or create an account.

```bash
plandex sign-in
```

Plandex will prompt you for all required information to sign in, accept an invite, or create an account.

### invite

Invite a user to join your org.

```bash
plandex invite # prompt for email, name, and role
plandex invite name@domain.com 'Full Name' member # invite with email, name, and role 
```

Users can be invited as `member`, `admin`, or `owner`.

### revoke

Revoke an invite or remove a user from your org.

```bash
plandex revoke # select from a list of users and invites
plandex revoke name@domain.com # by email
```

### users

List users and pending invites in your org.

```bash
plandex users
```

