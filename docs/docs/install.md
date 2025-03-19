---
sidebar_position: 1
sidebar_label: Install
---

# Install Plandex

## Quick Install

```bash
curl -sL https://plandex.ai/install.sh | bash
```

## Manual install

Grab the appropriate binary for your platform from the latest [release](https://github.com/plandex-ai/plandex/releases) and put it somewhere in your `PATH`.

## Build from source

```bash
git clone https://github.com/plandex-ai/plandex.git
git checkout v2
cd plandex/app/cli
go build -ldflags "-X plandex/version.Version=$(cat version.txt)"
mv plandex /usr/local/bin # adapt as needed for your system
```

## Windows

Windows is supported via [WSL](https://learn.microsoft.com/en-us/windows/wsl/about).

Plandex only works correctly in the WSL shell. It doesn't work in the Windows CMD prompt or PowerShell.

## Upgrading from v1 to v2

When you install the Plandex v2 CLI with the quick install script, it will rename your existing `plandex` command to `plandex1` (and the `pdx` alias to `pdx1`). Plandex v2 is designed to run *separately* from v1 rather than upgrading in place. [More details here.](./upgrading-v1-to-v2.md)