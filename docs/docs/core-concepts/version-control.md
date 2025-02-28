---
sidebar_position: 7
sidebar_label: Version Control
---

# Version Control

Just about every aspect of a Plandex plan is version-controlled, and anything that can happen during a plan creates a new version in the plan's history. This includes:

- Adding, removing, or updating context (when you do it manually or when Plandex does it automatically).
- When you send a prompt.
- When Plandex responds.
- When Plandex builds the plan's proposed updates to a file into a pending change.
- When pending changes are rejected.
- When pending changes are applied to your project.
- When models or model settings are updated.

## Viewing History

To see the history of your plan, use the `plandex log` command:

```bash
plandex log
```

## Rewinding

To rewind the plan to an earlier state, use the `plandex rewind` command:

```bash
plandex rewind # Select a previous state to rewind to
plandex rewind 3  # Rewind 3 steps
plandex rewind a7c8d66  # Rewind to a specific step
```

## Preventing History Loss With Branches

Note that currently, there's no way to undo a `rewind` and recover any history that may have been cleared as a result. That said, you can use `rewind` without losing any history with [branches](./branches.md). Use `plandex checkout` to a create a new branch before executing `rewind`, and the original branch will still include the history from before the `rewind`.

```bash
plandex checkout undo-changes # create a new branch called 'undo-changes'
plandex rewind ef883a # history is rewound in 'undo-changes' branch
plandex checkout main # main branch still retains original history
```

## Viewing Conversation

While the Plandex history includes an entry for each message in the conversation, message content isn't included. To see the full conversation history, use the `plandex convo` command:

```bash
plandex convo
```

## Rewinding After `plandex apply`

Like any other action that modifies a plan, running `plandex apply` to apply pending changes to your project file creates a new version in the plan's history. The `plandex apply` action can also be undone with `plandex rewind`.

While previous versions Plandex would not also revert the changes to your project files, this is now the default behavior as of v2.0.0. If there are potential conflicts (i.e. you've made changes on top since applying), Plandex will prompt you to decide how to handle the conflict.

This behavior can be disabled if desired by setting the `auto-revert-on-rewind` config setting to `false`:

```bash
plandex set-config auto-revert-on-rewind false
plandex set-config default auto-revert-on-rewind false # set the default value for all new plans
```
