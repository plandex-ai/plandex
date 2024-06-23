# Manual Install

If you prefer to install Plandex manually, follow these steps:

1. Download the appropriate binary for your platform from the [latest release](https://github.com/plandex-ai/plandex/releases).
2. Move the binary to a directory in your `PATH`. For example:

```bash
mv plandex /usr/local/bin
```

3. Verify the installation by running:

```bash
plandex --version
```

You should see the version number of Plandex printed in the terminal.

## Using the `pdx` Alias

For convenience, you can use the `pdx` alias instead of typing `plandex` for every command. You can set up the alias manually by adding the following line to your shell configuration file (e.g., `.bashrc`, `.zshrc`):

```bash
alias pdx='plandex'
```

After adding the alias, reload your shell configuration:

```bash
source ~/.bashrc  # or source ~/.zshrc
```

Now you can use the `pdx` alias:

```bash
pdx --version
```

This should also print the version number of Plandex.
