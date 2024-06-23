# Reviewing and Applying Changes

Plandex will stream the response to your terminal and build up a set of changes along the way. You can review and apply these changes to your project files.

## Reviewing Changes

To review the changes that Plandex has built up so far, use the `plandex changes` command:

```bash
plandex changes
```

This will open a user-friendly TUI changes viewer.

## Applying Changes

If you're happy with the changes, apply them to your files:

```bash
plandex apply
```

If you're in a git repo, Plandex will automatically add a commit with a nicely formatted message describing the changes. Any uncommitted changes that were present in your working directory beforehand will be unaffected.

## Using the `pdx` Alias

For convenience, you can use the `pdx` alias instead of typing `plandex` for every command:

```bash
pdx changes
pdx apply
```
