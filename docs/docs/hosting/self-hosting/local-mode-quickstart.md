---
sidebar_position: 1
sidebar_label: Local Mode Quickstart
---

# Self-Hosting

Plandex is open source and uses a client-server architecture. The server can be self-hosted. You can either run it locally or on a cloud server that you control. To run it on a cloud server, go to  [Advanced Self-Hosting](advanced-self-hosting.md) section. To run it locally, keep reading below.

## Local Mode Quickstart

The quickstart requires git, docker, and docker-compose. It's designed for local use with a single user. Note that for `macOS` the docker app must be installed, not the headless version. For detailed instructions, see [Installing Docker](./installing-docker.md).

1. Run the server in local mode: 

```bash
git clone https://github.com/plandex-ai/plandex.git
cd plandex/app
./start_local.sh
```

2. In a new terminal session, install the Plandex CLI if you haven't already:

```bash
curl -sL https://plandex.ai/install.sh | bash
```

3. Run:

```bash
plandex sign-in
```

4. When prompted 'Use Plandex Cloud or another host?', select 'Local mode host'. Confirm the default host, which is `http://localhost:8099`.

5. Decide on the model provider(s) you want to use. The quickest option is to use OpenRouter.ai, but you can also use [many other providers](https://docs.plandex.ai/models/model-providers).

If you're using OpenRouter.ai, first [sign up here.](https://openrouter.ai/signup) Then [generate an API key here.](https://openrouter.ai/keys) Set the `OPENROUTER_API_KEY` environment variable:

```bash
export OPENROUTER_API_KEY=...
```

6. In a project directory, start the Plandex REPL:

```bash
plandex
```

You're ready to start building!

## Upgrade

To upgrade after a new release, just use `ctrl-c` to stop the server, then run the script again:

```bash
./start_local.sh
```

The script will pull the latest image before the server starts.


