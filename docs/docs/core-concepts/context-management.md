# Context Management

Context in Plandex refers to the files, directories, URLs, and other data that the AI uses to understand and work on your project.

## Loading Context

To load files or directories into context, use the `plandex load` command:

```bash
plandex load file1.ts file2.ts
plandex load src -r  # Load a directory recursively
```

## Viewing Context

To list everything in context, use the `plandex ls` command:

```bash
plandex ls
```

## Removing Context

To remove context, use the `plandex rm` command:

```bash
plandex rm file1.ts
```

## Clearing Context

To clear all context, use the `plandex clear` command:

```bash
plandex clear
```

## Updating Context

To update outdated context, use the `plandex update` command:

```bash
plandex update
```
