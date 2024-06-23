# CLI Reference

## Key Commands

- `plandex new`: Start a new plan
- `plandex load`: Load files, directories, URLs, notes, or piped data into context
- `plandex tell`: Describe a task, ask a question, or chat
- `plandex changes`: Review pending changes in a TUI
- `plandex diff`: Review pending changes in 'git diff' format
- `plandex apply`: Apply pending changes to project files
- `plandex reject`: Reject pending changes to one or more project files

## Plans

- `plandex new`: Start a new plan
- `plandex plans`: List plans
- `plandex cd`: Set current plan by name or index
- `plandex current`: Show current plan
- `plandex delete-plan`: Delete plan by name or index
- `plandex rename`: Rename the current plan
- `plandex archive`: Archive a plan
- `plandex plans --archived`: List archived plans
- `plandex unarchive`: Unarchive a plan

## Context

- `plandex load`: Load files, directories, URLs, notes, or piped data into context
- `plandex ls`: List everything in context
- `plandex rm`: Remove context by index, range, name, or glob
- `plandex update`: Update outdated context
- `plandex clear`: Remove all context

## Branches

- `plandex branches`: List plan branches
- `plandex checkout`: Checkout or create a branch
- `plandex delete-branch`: Delete a branch by name or index

## History

- `plandex log`: Show log of plan updates
- `plandex rewind`: Rewind to a previous state
- `plandex convo`: Show plan conversation
- `plandex convo --plain`: Show conversation in plain text

## Control

- `plandex tell`: Describe a task, ask a question, or chat
- `plandex continue`: Continue the plan
- `plandex build`: Build any pending changes

## Streams

- `plandex ps`: List active and recently finished plan streams
- `plandex connect`: Connect to an active plan stream
- `plandex stop`: Stop an active plan stream

## AI Models

- `plandex models`: Show current plan model settings
- `plandex models default`: Show org-wide default model settings for new plans
- `plandex models available`: Show all available models
- `plandex set-model`: Update current plan model settings
- `plandex set-model default`: Update org-wide default model settings for new plans
- `plandex models add`: Add a custom model
- `plandex models delete`: Delete a custom model
- `plandex model-packs`: Show all available model packs
- `plandex model-packs create`: Create a new custom model pack
- `plandex model-packs delete`: Delete a custom model pack

## Accounts

- `plandex sign-in`: Sign in, accept an invite, or create an account
- `plandex invite`: Invite a user to join your org
- `plandex revoke`: Revoke an invite or remove a user from your org
- `plandex users`: List users and pending invites in your org
