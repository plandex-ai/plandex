---
sidebar_position: 3
sidebar_label: Context
---

# Context Management

Context in Plandex refers to files, directories, URLs, images, notes, or piped in data that the LLM uses to understand and work on your project. Context is always associated with a [plan](./plans.md).

Changes to context are [version controlled](./version-control.md) and can be [branched](./branches.md).

## Automatic vs. Manual

As of v2, Plandex loads context automatically by default. When a new plan is created, a [project map](#loading-project-maps) is generated and loaded into context. The LLM then uses this map to select relevant context before planning a task or responding to a message.

### Tradeoffs

Automatic context loading makes Plandex more powerful and easier to work with, but there are tradeoffs in terms of cost, focus, and output speed. If you're trying to minimize costs or you know for sure that only one or two files are relevant to a task, you might prefer to load context manually.

### Setting Manual Mode

You can use manual context loading by:

- Using `set-auto` to choose a lower [autonomy level](./autonomy.md) that has auto-load-context disabled (like `plus` or `basic`).

```bash
plandex set-auto plus
plandex set-auto basic
plandex set-auto default plus # set the default value for all new plans
```

- Starting a new REPL or a new plan with the `--plus` or `--basic` flags, which will automatically set the config option to the chosen autonomy level.

```bash
plandex --plus
plandex new --basic
```

- Setting the `auto-load-context` [config option](./configuration.md) to `false`:

```bash
plandex set-config auto-load-context false
plandex set-config default auto-load-context false # set the default value for all new plans
```

### Smart Context Window Management

Another new context management feature in v2 is smart context window management. When making a plan with multiple steps, Plandex will determine which files are relevant to each step. Only those files will be loaded into context during implementation.

When combined with automatic context loading, this effectively creates a sliding context window that grows and shrinks as needed throughout the plan.

Smart context can also be used when you're managing context manually. To give an example: say you've manually loaded a directory with 10 files in it, and you need to make some updates to each one of them. Without smart context, each step of the implementation will load all 10 files into context. But if you use smart context, only the one or two files that are edited in each step will be loaded.

Smart context is enabled in the `plus` autonomy level and above. You can also toggle it with `set-config`:

```bash
plandex set-config smart-context true
plandex set-config smart-context false
plandex set-config default smart-context false # set the default value for all new plans
```

### Automatic Context Updates

When you make your own changes to files in context separately from Plandex, those files need to be updated before the plan can continue. Previously, Plandex would prompt you to update context every time a file was changed. This is now automatic by default.

Automatic updates are enabled in the `plus` autonomy level and above. You can also toggle them with `set-config`:

```bash
plandex set-config auto-update-context true
plandex set-config auto-update-context false
plandex set-config default auto-update-context false # set the default value for all new plans
```

### Autonomy Matrix

Here are the different autonomy levels as they relate to context management config options:

|                       | `none` | `basic` | `plus` | `semi` | `full` |
| --------------------- | ------ | ------- | ------ | ------ | ------ |
| `auto-load-context`   | ❌     | ❌      | ❌     | ✅     | ✅     |
| `smart-context`       | ❌     | ❌      | ✅     | ✅     | ✅     |
| `auto-update-context` | ❌     | ❌      | ✅     | ✅     | ✅     |

### Mixing Automatic and Manual Context

You can manually load additional context even if automatic loading is enabled. The way this additional context is handled works somewhat differently.

First, consider how automatic context loading works across each stage of a plan:

#### Automatic context loading (no manual context added)

1. **Context loading:** Only the project map is initially loaded. The map, along with your prompt, is used to select relevant context.
2. **Planning:** Only context selected in step 1 is loaded.
3. **Implementation:** Smart context (if enabled) filters context again, loading only what's directly relevant to each step.

Here's how it changes when you load manual context on top:

#### Automatic loading + manual context

1. **Context loading:** Your manually loaded context is **always included** alongside the project map.
2. **Planning:** Manually loaded context is always loaded, whether or not it's selected by the map-based context selection step.
3. **Implementation:** Smart context (if enabled) filters all context again (both manual and automatic), loading only what's directly relevant to each implementation step.

Loading files manually when using automatic context loading can sometimes be useful when you **know** certain files are relevant and don't want to risk the LLM leaving them out, or when the LLM is struggling to select the right context. If there are files that can help the LLM select the right context, like READMEs or documentation that describes the structure of the project, those can also be good candidates for manual loading.

Another use for manual context loading is for context types that can't be loaded automatically, like URLs, notes, or piped data (for now Plandex can only automatically load project files and images within the project).

## Manually Loading Context

To load files, directories, directory layouts, urls, images, notes, or piped data into a plan's context, use the `plandex load` command.

### Loading Files

You can pass `load` one or more file paths. File paths are relative to the current directory in your terminal.

```bash
plandex load component.ts # single file
plandex load component.ts action.ts reducer.ts # multiple files
pdx l component.ts # alias
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

### Loading Files and Directories in the REPL

In the [Plandex REPL](../repl.md), you can use the shortcut `@` plus a relative file path to load a file or directory into context.

```bash
@component.ts # loads component.ts
@lib # loads lib directory, and all its files and subdirectories
```

### Loading Directory Layouts

There are tasks where it's helpful for the LLM to the know the structure of your project or sections of your project, but it doesn't necessarily need to the see the content of every file. In that case, you can pass in a directory with the `--tree` flag to load in the directory layout. It will include just the names of all included files and subdirectories (and each subdirectory's files and subdirectories, and so on).

```bash
plandex load . --tree # loads the layout of the current directory and its subdirectories (file names only)
plandex load src/components --tree # loads the layout of the src/components directory
```

### Loading Project Maps

Plandex can create a **project map** for any directory using [tree-sitter](https://tree-sitter.github.io/tree-sitter). This shows all the top-level symbols, like variables, functions, classes, etc. in each file. 30+ languages are supported. For non-supported languages, files are still listed without symbols so that the model is aware of their existence.

Maps are mainly used for selecting context during automatic context loading, but can also be used with manual context management in order to improve output. Maps make it much more likely that an LLM will, for example, use an existing function in your project (and call it correctly) rather than generating a new one that does the same thing.

```bash
plandex load . --map
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

For most models that support images, png, jpeg, non-animated gif, and webp formats are supported. Some models may support fewer or additional formats.

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

You can also see the content of any context item with the `plandex show` command:

```bash
plandex show component.ts # show the content of component.ts
plandex show 2 # show the content of the second item in the `plandex ls` list
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

If files, directory layouts, or URLs in context are modified outside of Plandex, they will need to be updated next time you send a prompt.

Whether they'll be updated automatically or you'll be prompted to update them depends on the `auto-update-context` config option.

You can also update any outdated files with the `update` command.

```bash
plandex update # update files in context
```
