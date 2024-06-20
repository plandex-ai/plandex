# Context Management

Plandex allows you to efficiently manage context in the terminal. You can load files, directories, URLs, and notes into the plan context.

## Loading Context

To load context, use the `plandex load` command:

```bash
plandex load component.ts action.ts reducer.ts
plandex load lib -r
plandex load https://redux.js.org/usage/writing-tests
```

## Listing and Removing Context

You can list and remove context using the following commands:

```bash
plandex ls
plandex rm [context-name]
plandex clear
```
