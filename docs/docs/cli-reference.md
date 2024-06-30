---
sidebar_position: 6
sidebar_label: CLI Reference
---

# Command List

This section provides a comprehensive list of all Plandex CLI commands.

## General Commands

- `plandex new`: Start a new plan.
- `plandex load`: Load files, directories, URLs, notes, or piped data into context.
- `plandex tell`: Describe a task, ask a question, or chat.
- `plandex changes`: Review pending changes in a TUI.
- `plandex diff`: Review pending changes in 'git diff' format.
- `plandex apply`: Apply pending changes to project files.
- `plandex reject`: Reject pending changes to one or more project files.

## Plan Management

- `plandex plans`: List plans.
- `plandex cd`: Set current plan by name or index.
- `plandex current`: Show current plan.
- `plandex delete-plan`: Delete a plan by name or index.
- `plandex rename`: Rename the current plan.
- `plandex archive`: Archive a plan.
- `plandex unarchive`: Unarchive a plan.

## Context Management

- `plandex ls`: List everything in context.
- `plandex rm`: Remove context by index, range, name, or glob.
- `plandex update`: Update outdated context.
- `plandex clear`: Remove all context.

## Branch Management

- `plandex branches`: List plan branches.
- `plandex checkout`: Checkout or create a branch.
- `plandex delete-branch`: Delete a branch by name or index.

## History and Rewind

- `plandex log`: Show plan history.
- `plandex rewind`: Rewind to a previous state.
- `plandex convo`: Show plan conversation.

## Background Tasks

- `plandex ps`: List active and recently finished plan streams.
- `plandex connect`: Connect to an active plan stream.
- `plandex stop`: Stop an active plan stream.

## Model Management

- `plandex models`: Show plan model settings.
- `plandex models default`: Show org-wide default model settings for new plans.
- `plandex models available`: Show all available models.
- `plandex set-model`: Update current plan model settings.
- `plandex set-model default`: Update org-wide default model settings for new plans.
- `plandex models add`: Add a custom model.
- `plandex models delete`: Delete a custom model.
- `plandex model-packs`: Show all available model packs.
- `plandex model-packs create`: Create a new custom model pack.
- `plandex model-packs delete`: Delete a custom model pack.

## Account Management

- `plandex sign-in`: Sign in, accept an invite, or create an account.
- `plandex invite`: Invite a user to join your org.
- `plandex revoke`: Revoke an invite or remove a user from your org.
- `plandex users`: List users and pending invites in your org.
