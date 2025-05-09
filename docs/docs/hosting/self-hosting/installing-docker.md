---
sidebar_position: 3
sidebar_label: Installing Docker
---

# Installing Docker

Plandex uses docker to deploy and run a local server. Please refer to the instructions for your operating system.

## macOS

1. Install docker (cask) using Homebrew:

```bash
brew install --cask docker
```

2. Launch the Docker app in macOS and proceed through the initial setup.

3. When setup is concluded, you may now run the local server as in the [Local Mode Quickstart](./local-mode-quickstart.md)

Note: The headless install using `brew install docker docker-compose` (without the `--cask` flag) will cause an error when trying to run the local server, possibly indicating that `docker compose` is an unknown command. It's not required to install `docker-compose` separately, as the cask contains this functionality.
