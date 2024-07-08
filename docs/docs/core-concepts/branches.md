---
sidebar_position: 6
sidebar_label: Branches
---

# Branches

Branches in Plandex allow you to easily try out multiple approaches to a task and see which gives you the best results. They work in conjunction with [version control](./version-control.md). Use cases include:

- Comparing different prompting strategies.
- Comparing results with different files in context.
- Comparing results with different models or model-settings.
- Using `plandex rewind` without losing history (first check out a new branch, then rewind).

## Creating a Branch

To create a new branch, use the `plandex checkout` command:

```bash
plandex checkout new-branch
```

## Switching Branches

To switch to a different branch, also use the `plandex checkout` command:

```bash
plandex checkout existing-branch
```

## Listing Branches

To list all branches, use the `plandex branches` command:

```bash
plandex branches
```

## Deleting a Branch

To delete a branch, use the `plandex delete-branch` command:

```bash
plandex delete-branch branch-name
```
