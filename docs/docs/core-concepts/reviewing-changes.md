---
sidebar_position: 4
sidebar_label: Pending Changes
---

# Pending Changes

When you give Plandex a task, the changes aren't applied directly to your project files. Instead, they are accumulated in Plandex's version-controlled **sandbox** so that you can review them first.

## `plandex diffs` / Changes TUI

When Plandex has finished with your task, you can review the proposed changes with the `plandex diff` command, which shows them in `git diff` format:

```bash
plandex diff
```

`--plain/-p`: Outputs the conversation in plain text with no ANSI codes.

You can also view them in Plandex's changes TUI:

```bash
plandex changes
```

## Rejecting Files

While we're working hard to make file updates as reliable as possible, bad updates can still happen. If the plan's changes were applied incorrectly to a file, you can either [apply the changes](#apply-the-changes) and then fix the problems manually, *or* you can reject the updates to that file and then make the proposed changes yourself manually. 

To reject changes to a file (or multiple files), you can run `plandex reject` with one ore more file paths:

```bash
plandex reject 
```

You can also reject changes using the `r` hotkey in the `plandex changes` TUI.

Once the bad update is rejected, copy the changes from the plan's output or run `plandex convo` to output the full conversation and copy them from there. Then apply the updates to that file yourself.

## Apply The Changes

Once you're happy with the plan's changes, you can apply them to your project files with `plandex apply`:

```bash
plandex apply
```

If you're in a git repository, Plandex will give you the option of grouping the changes into a git commit with an automatically generated commit message. Any uncommitted changes that were present in your working directory beforehand will be unaffected.

You can skip the `plandex apply` confirmation with the `-y` flag.