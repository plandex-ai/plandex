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

## REPL

The easiest way to use Plandex is through the REPL. Start it in your project directory with:

```bash
plandex
```

or for short:

```bash
pdx
```

### Flags

The REPL has a few convenient flags you can use to start it with different modes, autonomy settings, and model packs. You can pass any of these to `plandex` or `pdx` when starting the REPL.

```
  Mode
    --chat, -c     Start in chat mode (for conversation without making changes)
    --tell, -t     Start in tell mode (for implementation)

  Autonomy
    --no-auto      None → step-by-step, no automation
    --basic        Basic → auto-continue plans, no other automation
    --plus         Plus → auto-update context, smart context, auto-commit changes
    --semi         Semi-Auto → auto-load context
    --full         Full-Auto → auto-apply, auto-exec, auto-debug

  Models
    --daily        Daily driver pack (default models, balanced capability, cost, and speed)
    --reasoning    Similar to daily driver, but uses reasoning model for planning
    --strong       Strong pack (more capable models, higher cost and slower)
    --cheap        Cheap pack (less capable models, lower cost and faster)
    --oss          Open source pack (open source models)
    
    --gemini-planner       Gemini pack (Gemini 2.5 Pro for planning, default models for other roles)
    --o3-planner           OpenAI o3-medium for planning, default models for other roles
    --r1-planner           DeepSeek R1 for planning, default models for other roles
    --perplexity-planner   Perplexity for planning, default models for other roles
    --opus-planner         Anthropic Opus 4 for planning, default models for other roles
```

All commands listed below can be run in the REPL by prefixing them with a backslash (`\`), e.g. `\new`.

## Plans

### new

Start a new plan.

```bash
plandex new
plandex new -n new-plan # with name
```

`--name/-n`: Name of the new plan. The name is generated automatically after first prompt if no name is specified on creation.

`--context-dir/-d`: Base directory to load context from when auto-loading context is enabled. Defaults to `.` (current directory). Set a different directoy if you don't want all files to be included in the project map.

`--no-auto`: Start the plan with auto-mode 'None' (step-by-step, no automation).

`--basic`: Start the plan with auto-mode 'Basic' (auto-continue plans, no other automation).

`--plus`: Start the plan with auto-mode 'Plus' (auto-update context, smart context, auto-commit changes).

`--semi`: Start the plan with auto-mode 'Semi-Auto' (auto-load context).

`--full`: Start the plan with auto-mode 'Full-Auto' (auto-apply, auto-exec, auto-debug).

`--daily`: Start the plan with the daily driver model pack.

`--reasoning`: Start the plan with the reasoning model pack.

`--strong`: Start the plan with the strong model pack.

`--cheap`: Start the plan with the cheap model pack.

`--oss`: Start the plan with the open source model pack.

`--gemini-planner`: Start the plan with the Gemini planner model pack.

`--o3-planner`: Start the plan with the OpenAI o3-medium planner model pack.

`--r1-planner`: Start the plan with the DeepSeek R1 planner model pack.

`--perplexity-planner`: Start the plan with the Perplexity planner model pack.

`--opus-planner`: Start the plan with the Anthropic Opus 4 planner model pack.

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

Delete a plan by name, index, range, pattern, or select from a list.

```bash
plandex delete-plan # select from a list of plans
plandex delete-plan some-plan # by name
plandex delete-plan 4 # by index in `plandex plans`
plandex delete-plan 2-4 # by range of indices
plandex delete-plan 'docs-*' # by pattern
plandex delete-plan --all # delete all plans
pdx dp # alias
```

`--all/-a`: Delete all plans.

### rename

Rename the current plan.

```bash
plandex rename # prompt for new name
plandex rename new-name # set new name
```

### archive

Archive a plan.

```bash
plandex archive # select from a list of plans
plandex archive some-plan # by name
plandex archive 4 # by index in `plandex plans`

pdx arc # alias
```

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

`--map`: Load file map of the given directory (function/method/class signatures, variable names, types, etc.)

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
plandex rm 2-4 # by range of indices

plandex remove # longer alias
plandex unload # longer alias
```

### show

Output context by name or index.

```bash
plandex show some-file.ts # by name
plandex show 4 # by index in `plandex ls`
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

Describe a task.

```bash
plandex tell -f prompt.txt # from file
plandex tell # open vim to write prompt
plandex tell "add a cancel button to the left of the submit button" # inline

pdx t # alias
```

`--file/-f`: File path containing prompt.

`--stop/-s`: Stop after a single model response (don't auto-continue). Defaults to opposite of config value `auto-continue`.

`--no-build/-n`: Don't build proposed changes into pending file updates. Defaults to opposite of config value `auto-build`.

`--bg`: Run task in the background. Only allowed if `--auto-load-context` and `--apply/-a` are not enabled. Not allowed with the default [autonomy level](./core-concepts/autonomy.md) in Plandex v2.

`--auto-update-context`: Automatically confirm context updates. Defaults to config value `auto-update-context`.

`--auto-load-context`: Automatically load context using project map. Defaults to config value `auto-load-context`.

`--smart-context`: Use smart context to only load the necessary file(s) for each step during implementation. Defaults to config value `smart-context`.

`--no-exec`: Don't execute commands after successful apply. Defaults to opposite of config value `can-exec`.

`--auto-exec`: Automatically execute commands after successful apply without confirmation. Defaults to config value `auto-exec`.

`--skip-menu`: Skip interactive menu when response finishes and changes are pending. Defaults to config value `skip-changes-menu`.

`--debug`: Automatically execute and debug failing commands (optionally specify number of tries—default is 5). Defaults to config values of `auto-debug` and `auto-debug-tries`.

`--apply/-a`: Automatically apply changes (and confirm context updates). Defaults to config value `auto-apply`.

`--commit/-c`: Commit changes to git when `--apply/-a` is passed. Defaults to config value `auto-commit`.

`--skip-commit`: Don't commit changes to git. Defaults to opposite of config value `auto-commit`.

### continue

Continue the plan.

```bash
plandex continue

pdx c # alias
```

`--stop/-s`: Stop after a single model response (don't auto-continue). Defaults to opposite of config value `auto-continue`.

`--no-build/-n`: Don't build proposed changes into pending file updates. Defaults to opposite of config value `auto-build`.

`--bg`: Run task in the background. Only allowed if `--auto-load-context` and `--apply/-a` are not enabled. Not allowed with the default [autonomy level](./core-concepts/autonomy.md) in Plandex v2.

`--auto-update-context`: Automatically confirm context updates. Defaults to config value `auto-update-context`.

`--auto-load-context`: Automatically load context using project map. Defaults to config value `auto-load-context`.

`--smart-context`: Use smart context to only load the necessary file(s) for each step during implementation. Defaults to config value `smart-context`.

`--no-exec`: Don't execute commands after successful apply. Defaults to opposite of config value `can-exec`.

`--auto-exec`: Automatically execute commands after successful apply without confirmation. Defaults to config value `auto-exec`.

`--skip-menu`: Skip interactive menu when response finishes and changes are pending. Defaults to config value `skip-changes-menu`.

`--debug`: Automatically execute and debug failing commands (optionally specify number of tries—default is 5). Defaults to config values of `auto-debug` and `auto-debug-tries`.

`--apply/-a`: Automatically apply changes (and confirm context updates). Defaults to config value `auto-apply`.

`--commit/-c`: Commit changes to git when `--apply/-a` is passed. Defaults to config value `auto-commit`.

`--skip-commit`: Don't commit changes to git. Defaults to opposite of config value `auto-commit`.

### build

Build any unbuilt pending changes from the plan conversation.

```bash
plandex build
pdx b # alias
```

`--bg`: Build in the background. Not allowed if `--apply/-a` is enabled.

`--stop/-s`: Stop after a single model response (don't auto-continue). Defaults to opposite of config value `auto-continue`.

`--no-build/-n`: Don't build proposed changes into pending file updates. Defaults to opposite of config value `auto-build`.

`--auto-update-context`: Automatically confirm context updates. Defaults to config value `auto-update-context`.

`--no-exec`: Don't execute commands after successful apply. Defaults to opposite of config value `can-exec`.

`--auto-exec`: Automatically execute commands after successful apply without confirmation. Defaults to config value `auto-exec`.

`--skip-menu`: Skip interactive menu when response finishes and changes are pending. Defaults to config value `skip-changes-menu`.

`--debug`: Automatically execute and debug failing commands (optionally specify number of tries—default is 5). Defaults to config values of `auto-debug` and `auto-debug-tries`.

`--apply/-a`: Automatically apply changes (and confirm context updates). Defaults to config value `auto-apply`.

`--commit/-c`: Commit changes to git when `--apply/-a` is passed. Defaults to config value `auto-commit`.

`--skip-commit`: Don't commit changes to git. Defaults to opposite of config value `auto-commit`.

### chat

Ask a question or chat without making any changes.

```bash
plandex chat "is it clear from the context how to add a new line chart?"
pdx ch # alias
```

`--file/-f`: File path containing prompt.

`--bg`: Run task in the background. Not allowed if `--auto-load-context` is enabled. Not allowed with the default [autonomy level](./core-concepts/autonomy.md) in Plandex v2.

`--auto-update-context`: Automatically confirm context updates. Defaults to config value `auto-update-context`.

`--auto-load-context`: Automatically load context using project map. Defaults to config value `auto-load-context`.

### debug

Repeatedly run a command and automatically attempt fixes until it succeeds, rolling back changes on failure. Defaults to 5 tries before giving up.

```bash
plandex debug 'npm test' # try 5 times or until it succeeds
plandex debug 10 'npm test' # try 10 times or until it succeeds
pdx db 'npm test' # alias
```

`--commit/-c`: Commit changes to git when `--apply/-a` is passed. Defaults to config value `auto-commit`.

`--skip-commit`: Don't commit changes to git. Defaults to opposite of config value `auto-commit`.

## Changes

### diff

Review pending changes in 'git diff' format or in a local browser UI.

```bash
plandex diff
plandex diff --ui
```

`--plain/-p`: Output diffs in plain text with no ANSI codes.

`--ui/-u`: Review pending changes in a local browser UI.

`--side-by-side/-s`: Show diffs UI in side-by-side view

`--line-by-line/-l`: Show diffs UI in line-by-line view

### apply

Apply pending changes to project files.

```bash
plandex apply
pdx ap # alias
```

`--auto-update-context`: Automatically confirm context updates. Defaults to config value `auto-update-context`.

`--no-exec`: Don't execute commands after successful apply. Defaults to opposite of config value `can-exec`.

`--auto-exec`: Automatically execute commands after successful apply without confirmation. Defaults to config value `auto-exec`.

`--debug`: Automatically execute and debug failing commands (optionally specify number of tries—default is 5). Defaults to config values of `auto-debug` and `auto-debug-tries`.

`--commit/-c`: Commit changes to git when `--apply/-a` is passed. Defaults to config value `auto-commit`.

`--skip-commit`: Don't commit changes to git. Defaults to opposite of config value `auto-commit`.

`--full`: Apply the plan and debug in full auto mode.

### reject

Reject pending changes to one or more project files.

```bash
plandex reject # select from a list of pending files to reject
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
plandex rewind # select from a list of previous states to rewind to
plandex rewind 3 # rewind 3 steps
plandex rewind a7c8d66 # rewind to a specific step from `plandex log`
```

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
plandex checkout some-branch -y # checkout by name or create a new branch with that name, auto-confirming branch creation

pdx co # alias
```

`--yes/-y`: Auto-confirm creating a new branch if it doesn't exist.

### delete-branch

Delete a branch by name or index.

```bash
plandex delete-branch # select from a list of branches
plandex delete-branch some-branch # by name
plandex delete-branch 4 # by index in `plandex branches`

pdx dlb # alias
```

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
pdx conn # alias
```

### stop

Stop an active plan stream.

```bash
plandex stop # select from a list of active streams
plandex stop a4de # by stream ID in `plandex ps`
plandex stop some-plan main # by plan name and branch name
```

## Configuration

### config

Show current plan config. Output includes configuration settings for the plan, such as autonomy level, model settings, and other plan-specific options.

```bash
plandex config
```

### config default

Show the default config used for new plans. Output includes the default configuration settings that will be applied to newly created plans.

```bash
plandex config default
```

### set-config

Update configuration settings for the current plan.

```bash
plandex set-config # select from a list of config options
plandex set-config auto-context true # set a specific config option
```

With no arguments, Plandex prompts you to select from a list of config options.

With arguments, allows you to directly set specific configuration options for the current plan.

### set-config default

Update the default configuration settings for new plans.

```bash
plandex set-config default # select from a list of config options
plandex set-config default auto-mode basic # set a specific default config option
```

Works exactly the same as set-config above, but sets the default configuration for all new plans instead of only the current plan.

### set-auto

Update the auto-mode (autonomy level) for the current plan.

```bash
plandex set-auto # select from a list of auto-modes
plandex set-auto full # set to full automation
plandex set-auto semi # set to semi-auto mode
plandex set-auto plus # set to plus mode
plandex set-auto basic # set to basic mode
plandex set-auto none # set to none (step-by-step, no automation)
```

With no arguments, Plandex prompts you to select from a list of automation levels.

With one argument, Plandex sets the automation level directly to the specified value.

### set-auto default

Set the default auto-mode for new plans.

```bash
plandex set-auto default # select from a list of auto-modes
plandex set-auto default basic # set default to basic mode
```

Works exactly the same as set-auto above, but sets the default automation level for all new plans instead of only the current plan.

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

### models custom

Manage custom models, providers, and model packs via JSON file.

```bash
plandex models custom
plandex models custom --save # save changes from the default custom-models.json file to the server
plandex models custom --file /path/to/models.json # import custom models/providers/model-packs from a non-default JSON file
```

`--save`: Save changes from the JSON file to the server.

`--file/-f`: Path to non-default custom models JSON file.

Without `--save`, this command will:

- Create an example models file if none exists, or sync current server state to the file
- Open the file in your configured editor
- Prompt you to save changes when you return

With `--save`, it will skip opening the editor and sync changes from the JSON file to the server.

### models available

Show available models.

```bash
plandex models available # show all available models
plandex models available --custom # show available custom models only
```

`--custom`: Show available custom models only.

### providers

Show all available model providers.

```bash
plandex providers
plandex providers --custom # show custom providers only (not supported on Plandex Cloud)
```

`--custom/-c`: Show custom providers only (not supported on Plandex Cloud).

### set-model

Update current plan models or model settings.

```bash
plandex set-model # select from a list of model packs or edit via JSON
plandex set-model daily # set model pack by name
plandex set-model --json # edit plan's model settings via JSON file at default path
plandex set-model --save # save changes from the plan's model settings JSON file to the server
plandex set-model --json --file /path/to/settings.json # set plan's model settings from a JSON file at a non-default path
```

`--json`: Edit plan's model settings via JSON file at default path.

`--save`: Save changes from the plan's model settings JSON file to the server.

`--file`: Set plan's model settings from a JSON file at a non-default path.

With no arguments, Plandex prompts you to either select a model pack or edit settings via JSON.

When using JSON mode without `--save`, Plandex will:

- Write current settings to a JSON file
- Open it in your configured editor
- Apply the changes when you save and return

With `--save`, it will skip opening the editor and sync changes from the JSON file to the server.

Model pack shortcuts are still available:

```bash
plandex set-model daily
plandex set-model reasoning
plandex set-model strong
plandex set-model cheap
plandex set-model oss
plandex set-model gemini
```

### set-model default

Update org-wide default model settings for new plans.

```bash
plandex set-model default # select from a list of model packs or edit via JSON
plandex set-model default daily # set default model pack by name
plandex set-model default --json # edit default settings via JSON file at default path
plandex set-model default --save # save changes from the default model settings JSON file to the server
plandex set-model default --json --file /path/to/settings.json # set default model settings from a JSON file at a non-default path
```

Works exactly the same as `set-model` above, but sets the default model settings for all new plans instead of only the current plan.

### model-packs

Show all available model packs.

```bash
plandex model-packs # list built-in and custom model packs
plandex model-packs --custom # list custom model packs only
```

`--custom`: Show available custom (user-created) model packs only.


### model-packs show

Show a built-in or custom model pack's settings.

```bash
plandex model-packs show # select from a list of built-in and custom model packs
plandex model-packs show some-model-pack # by name
```

## Account Management

### sign-in

Sign in, accept an invite, or create an account.

```bash
plandex sign-in
```

`--pin`: Sign in with a pin from the Plandex Cloud web UI.

Unless you pass `--pin` (from the Plandex Cloud web UI), Plandex will prompt you for all required information to sign in, accept an invite, or create an account.

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

## Integrations

### connect-claude

Connect a Claude Pro or Max subscription. When Plandex calls Anthropic models, it will use your Claude subscription up to its quota.

```bash
plandex connect-claude
```

### disconnect-claude

```bash
plandex disconnect-claude
```

Disconnect your Claude Pro or Max subscription and clear credentials from your device. 

## Plandex Cloud

### billing

Show the billing settings page.

```bash
plandex billing
```

### usage

Show Plandex Cloud current balance and usage report. Includes recent spend, amount saved by input caching, a breakdown of spend by plan, category, and model, and a log of individual transactions with the `--log` flag.

Defaults to showing usage for the current session if you're using the REPL. Otherwise, defaults to showing usage for the day so far.

Requires **Integrated Models** mode.

```bash
plandex usage
```

`--today`: Show usage for the day so far.

`--month`: Show usage for the current billing month.

`--plan`: Show usage for the current plan.

`--log`: Show a log of individual transactions. Defaults to showing the log for the current session if you're using the REPL. Otherwise, defaults to showing the log for the day so far. Works with `--today`, `--month`, and `--plan` flags.

Flags for `usage --log`:

`--debits`: Show only debits in the log.

`--purchases`: Show only purchases in the log.

`--page-size/-s`: Number of transactions to display per page.

`--page/-p`: Page number to display.






