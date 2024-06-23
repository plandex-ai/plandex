# Common Issues

This section addresses common issues that users may encounter while using Plandex and provides solutions to resolve them.

## Installation Issues

### Problem: Installation Fails

**Solution**: Ensure you have the necessary permissions to install software on your system. If you are using a Unix-based system, try running the installation command with `sudo`:

```bash
sudo curl -sL https://plandex.ai/install.sh | bash
```

### Problem: Command Not Found

**Solution**: Ensure that the directory where Plandex is installed is in your `PATH`. You can add it to your `PATH` by adding the following line to your shell configuration file (e.g., `.bashrc`, `.zshrc`):

```bash
export PATH=$PATH:/path/to/plandex
```

## Context Management Issues

### Problem: Context Not Loading

**Solution**: Ensure that the files or directories you are trying to load exist and are accessible. Check for any typos in the file or directory names. Use the `plandex ls` command to verify the current context.

## Plan Management Issues

### Problem: Unable to Create a New Plan

**Solution**: Ensure that you are in the correct directory and have the necessary permissions to create files. Use the `plandex new` command to create a new plan.

## Model Management Issues

### Problem: Custom Model Not Working

**Solution**: Ensure that the custom model is compatible with Plandex and that you have provided the correct details during the setup. Use the `plandex models available --custom` command to list all custom models and verify their details.

## General Issues

### Problem: Command Not Working as Expected

**Solution**: Ensure that you are using the correct syntax and options for the command. Refer to the [CLI Reference](../cli-reference/command-list.md) for detailed usage instructions.

If you continue to experience issues, please reach out to the Plandex support team for further assistance.
