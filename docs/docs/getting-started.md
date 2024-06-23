# Getting Started

## Creating a New Plan

First, `cd` into your project's directory. Make a new directory first with `mkdir your-project-dir` if you're starting on a new project.

```bash
cd your-project-dir
```

Then start your first plan with `plandex new`.

```bash
plandex new
```

## Loading Context

After creating a plan, load any relevant files, directories, or URLs into the plan context.

```bash
plandex load component.ts action.ts reducer.ts
plandex load lib -r # loads lib and all its subdirectories
plandex load tests/**/*.ts # loads all .ts files in tests and its subdirectories
plandex load . --tree # loads the layout of the current directory and its subdirectories (file names only)
plandex load https://redux.js.org/usage/writing-tests # loads the text-only content of the URL
npm test | plandex load # loads the output of `npm test`
plandex load -n 'add logging statements to all the code you generate.' # load a note into context
```

## Sending a Prompt

Now give the AI a task to do.

```bash
# with no arguments, vim or nano will open and you can type your task there
plandex tell 
# you can pass a task directly as a string 
# press enter for line breaks while inside the quotes
plandex tell 'build another component like this one that displays foo adapters in the table rather than bar adapters.
quote> use the same layout and styles, but update the column headers and formatting to match the new data.
quote> add the needed reducer and action as well. write tests for the new code.' 
# load a task from a file
plandex tell --file task.txt # or -f task.txt
```
