# Tips and Tricks

This section provides useful tips and tricks to help you get the most out of Plandex.

## Using the `pdx` Alias

For convenience, use the `pdx` alias instead of typing `plandex` for every command. The alias is automatically set up during installation.

```bash
pdx --version
```

## Efficient Context Management

### Loading Multiple Files

You can load multiple files into context at once:

```bash
plandex load file1.ts file2.ts
```

### Loading Directories Recursively

To load a directory and its subdirectories, use the `-r` flag:

```bash
plandex load src -r
```

### Adding Notes

You can add notes to the context to provide additional information:

```bash
plandex load -n "This is a note."
```

## Managing Plans and Branches

### Creating and Switching Plans

Create a new plan and switch to it:

```bash
plandex new -n my-plan
plandex cd my-plan
```

### Creating and Switching Branches

Create a new branch and switch to it:

```bash
plandex checkout new-branch
```

## Reviewing and Applying Changes

### Reviewing Changes

Use the `plandex changes` command to review pending changes in a TUI:

```bash
plandex changes
```

### Applying Changes

Apply pending changes to your project files:

```bash
plandex apply
```

## Using Background Tasks

### Running Tasks in the Background

Run a task in the background using the `--bg` flag:

```bash
plandex tell --bg "Add a new feature"
```

### Listing Background Tasks

List active and recently finished background tasks:

```bash
plandex ps
```

## Customizing Model Settings

### Viewing Model Settings

View the current model settings:

```bash
plandex models
```

### Changing Model Settings

Change the model settings:

```bash
plandex set-model planner openai/gpt-4
```

By following these tips and tricks, you can use Plandex more efficiently and effectively.
