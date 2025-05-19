---
sidebar_position: 1
sidebar_label: Local Mode Quickstart
---

# Self-Hosting

Plandex is open source and uses a client-server architecture. The server can be self-hosted. You can either run it locally or on a cloud server that you control. To run it on a cloud server, go to  [Advanced Self-Hosting](advanced-self-hosting.md) section. To run it locally, keep reading below.

## Local Mode Quickstart

The quickstart requires git, docker, and docker-compose. It's designed for local use with a single user.

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

5. If you don't have an OpenRouter account, first [sign up here.](https://openrouter.ai/signup) Then [generate an API key here.](https://openrouter.ai/keys) Set the `OPENROUTER_API_KEY` environment variable:

```bash
export OPENROUTER_API_KEY=...
```

6. **Optional**: set a `OPENAI_API_KEY` environment variable if you want OpenAI models to use the OpenAI API directly instead of OpenRouter (for slightly lower latency and costs). This requires an [OpenAI account.](https://platform.openai.com/signup).

```bash
export OPENAI_API_KEY=...
```

7. In a project directory, start the Plandex REPL:

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


