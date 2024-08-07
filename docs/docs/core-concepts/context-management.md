---
sidebar_position: 2
sidebar_label: Context
---

# Context Management

Context in Plandex refers to files, directories, URLs, images, notes, or piped in data that the LLM uses to understand and work on your project. Context is always associated with a [plan](./plans.md)

Changes to context are [version controlled](./version-control.md) and can be [branched](./branches.md).

## Loading Context

To load files, directories, directory layouts, urls, images, notes, or piped data into a plan's context, use the `plandex load` command.

### Loading Files

You can pass `load` one or more file paths. File paths are relative to the current directory in your terminal.

```bash
plandex load component.ts # single file
plandex load component.ts action.ts reducer.ts # multiple files
```

You can also load multiple files using glob patterns:

```bash
plandex load tests/**/*.ts # loads all .ts files in 'tests' and its subdirectories
plandex load * # loads all files in the current directory
```

You can load context from parent or sibling directories if needed by using `..` in your load paths.

```bash
plandex load ../file.go # loads file.go from parent directory
plandex load ../sibling-dir/test.go # loads test.go from sibling directory
```

### Loading Directories

You can load an entire directory with the `--recursive/-r` flag:

```bash
plandex load lib -r # loads lib, all its files and all its subdirectories
plandex load * -r # loads all files in the current directory and all its subdirectories
```

### Loading Directory Layouts

There are tasks where it's helpful for the LLM to the know the structure of your project or sections of your project, but it doesn't necessarily need to the see the content of every file. In that case, you can pass in a directory with the `--tree` flag to load in the directory layout. It will include just the names of all included files and subdirectories (and each subdirectory's files and subdirectories, and so on).

```bash
plandex load . --tree # loads the layout of the current directory and its subdirectories (file names only)
plandex load src/components --tree # loads the layout of the src/components directory
```

### Loading URLs

Plandex can load the text content of URLs, which can be useful for adding relevant documentation, blog posts, discussions, and the like.

```bash
plandex load https://redux.js.org/usage/writing-tests # loads the text-only content of the url
```

### Loading Images

Plandex can load images into context.

```bash
plandex load ui-mockup.png
```

For the default GPT-4o model, png, jpeg, non-animated gif, and webp formats are supported. For other models, support for images in general, and particular formats specifically, will depend on the model.

### Loading Notes

You can add notes to context, which are just simple strings.

```bash
plandex load -n 'add logging statements to all the code you generate.' # load a note into context
```

Notes can be useful as 'sticky' explanations or instructions that will tend to have more prominence throughout a long conversation than normal prompts. That's because long conversations are summarized to stay below a token limit, which can cause some details from your prompts to be dropped along the way. This doesn't happen if you use notes.

### Piping Into Context

You can pipe the results of other commands into context:

```bash
npm test | plandex load # loads the output of `npm test`
```

### Ignoring files

If you're in a git repo, Plandex respects `.gitignore` and won't load any files that you're ignoring. You can also add a `.plandexignore` file with ignore patterns to any directory.

You can force Plandex to load ignored files with the `--force/-f` flag:

```bash
plandex load .env --force # loads the .env file even if it's in .gitignore or .plandexignore
```

## Viewing Context

To list everything in context, use the `plandex ls` command:

```bash
plandex ls
```

## Removing Context

To remove selectively remove context, use the `plandex rm` command:

```bash
plandex rm component.ts # remove by name
plandex rm 2 # remove by number in the `plandex ls` list
plandex rm 2-5 # remove a range of indices
plandex rm lib/**/*.js # remove by glob pattern
plandex rm lib # remove whole directory
```

## Clearing Context

To clear all context, use the `plandex clear` command:

```bash
plandex clear
```

## Updating Context

If files in context are modified outside of Plandex, you will be prompted to update them the next time you send a prompt. You can also update any outdated files with the `update` command.

```bash
plandex update # update files in context
```