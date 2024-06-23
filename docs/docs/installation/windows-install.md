# Windows Installation (WSL)

Plandex requires the Windows Subsystem for Linux (WSL) to run on Windows. Follow these steps to install Plandex on Windows:

## Install WSL

1. Open PowerShell as Administrator and run the following command to install WSL:

```powershell
wsl --install
```

2. Restart your computer if prompted.

3. Set up your Linux distribution by following the on-screen instructions.

## Install Plandex

1. Open your WSL terminal (e.g., Ubuntu).

2. Run the following command to install Plandex:

```bash
curl -sL https://plandex.ai/install.sh | bash
```

3. Verify the installation by running:

```bash
plandex --version
```

You should see the version number of Plandex printed in the terminal.

## Using the `pdx` Alias

For convenience, you can use the `pdx` alias instead of typing `plandex` for every command. The alias is automatically set up during installation.

```bash
pdx --version
```

This should also print the version number of Plandex.

## Note

Plandex will not work in the Windows CMD shell. You must use the WSL terminal to run Plandex commands.
