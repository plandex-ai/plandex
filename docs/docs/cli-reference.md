# CLI Reference

## General Commands

- `plandex new`: Start a new plan.
- `plandex load [files-or-urls...]`: Load context from various inputs.
- `plandex tell [prompt]`: Send a prompt for the current plan.
- `plandex changes`: View, copy, or manage changes for the current plan.
- `plandex apply`: Apply a plan to the project.
- `plandex reject [files...]`: Reject pending changes.

## Plan Management

- `plandex plans`: List plans.
- `plandex cd [name-or-index]`: Set current plan by name or index.
- `plandex delete-plan [name-or-index]`: Delete a plan by name or index.
- `plandex archive [name-or-index]`: Archive a plan.
- `plandex unarchive [name-or-index]`: Unarchive a plan.

## Context Management

- `plandex ls`: List everything in context.
- `plandex rm [context-name]`: Remove context by name.
- `plandex clear`: Clear all context.

## Branch Management

- `plandex branches`: List plan branches.
- `plandex checkout [branch-name]`: Checkout an existing plan branch or create a new one.
- `plandex delete-branch [branch-name]`: Delete a plan branch by name or index.

## Model Management

- `plandex models`: Show plan model settings.
- `plandex set-model [role] [property] [value]`: Update current plan model settings.
- `plandex model-packs`: List all model packs.
- `plandex model-packs create`: Create a model pack.
- `plandex model-packs delete [pack-name]`: Delete a model pack by name or index.
