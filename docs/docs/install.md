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
cd plandex/app/cli
go build -ldflags "-X plandex/version.Version=$(cat version.txt)"
mv plandex /usr/local/bin # adapt as needed for your system
```

## Windows

Windows is supported via [WSL](https://learn.microsoft.com/en-us/windows/wsl/about).

Plandex only works correctly in the WSL shell. It doesn't work in the Windows CMD prompt or PowerShell.