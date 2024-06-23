# Installation

## Quick Install

```bash
curl -sL https://plandex.ai/install.sh | bash
```

## Manual Install

Grab the appropriate binary for your platform from the latest [release](https://github.com/plandex-ai/plandex/releases) and put it somewhere in your `PATH`.

## Build from Source

```bash
git clone https://github.com/plandex-ai/plandex.git
git clone https://github.com/plandex-ai/survey.git
cd plandex/app/cli
go build -ldflags "-X plandex/version.Version=$(cat version.txt)"
mv plandex /usr/local/bin # adapt as needed for your system
```

## Windows

Windows is supported via [WSL](https://learn.microsoft.com/en-us/windows/wsl/about). Note that Plandex won't work in the Windows CMD shell. The WSL terminal is required.

```bash
# Install WSL
wsl --install

# Install Plandex
curl -sL https://plandex.ai/install.sh | bash
```
