# Sending Prompts

After loading context, you can send a prompt to Plandex to describe a task, ask a question, or chat.

## Sending a Prompt

To send a prompt, use the `plandex tell` command:

```bash
plandex tell "Add a new feature to the project."
```

## Using a File

You can also load a prompt from a file:

```bash
plandex tell -f prompt.txt
```

## Using the `pdx` Alias

For convenience, you can use the `pdx` alias instead of typing `plandex` for every command:

```bash
pdx tell "Add a new feature to the project."
```
