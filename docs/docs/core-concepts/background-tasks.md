---
sidebar_position: 12
sidebar_label: Background Tasks
---

# Background Tasks

Plandex allows you to run tasks in the background, helping you work on multiple tasks in parallel.

**Note:** in Plandex v2, sending tasks to the background is disabled by default, because it's not compatible with automatic context loading. If you set a lower [autonomy level](./autonomy.md), you can use background tasks.

## Running a Task in the Background

To run a task in the background, use the `--bg` flag with `plandex tell` or `plandex continue`.

```bash
plandex tell --bg "Add an update credit card form to 'src/components/billing'"
plandex continue --bg
```

The plandex stream TUI also has a `b` hotkey that allows you to send a streaming plan to the background.

## Listing Background Tasks

To list active and recently finished background tasks, use the `plandex ps` command:

```bash
plandex ps
```

## Connecting to a Background Task

To connect to a running background task and view its stream in the plan stream TUI, use the `plandex connect` command:

```bash
plandex connect
```

## Stopping a Background Task

To stop a running background task, use the `plandex stop` command:

```bash
plandex stop
```
