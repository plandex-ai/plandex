---
sidebar_position: 3
sidebar_label: Quickstart
---

# Quickstart

## Install Plandex

```bash
curl -sL https://plandex.ai/install.sh | bash
```

[Click here for more installation options.](./install.md)

Note that Windows is supported via [WSL](https://learn.microsoft.com/en-us/windows/wsl/about). Plandex only works correctly on Windows in the WSL shell. It doesn't work in the Windows CMD prompt or PowerShell.

## Hosting Options

| Option  | Description |
|---------|--------------------------------|
| **Plandex Cloud (Integrated Models)** | • No separate accounts or API keys.<br/>• Easy multi-device usage.<br/>• Centralized billing and budgeting.<br/>• Quickest way to [get started.](https://app.plandex.ai/start?modelsMode=integrated)  |
| **Plandex Cloud (BYO API Key)** | • Use Plandex Cloud with your own [OpenRouter.ai](https://openrouter.ai) and [OpenAI](https://platform.openai.com) keys.<br/> |
| **Self-hosted/Local Mode** | • Run Plandex locally with Docker or host on your own server.<br/>• Use your own [OpenRouter.ai](https://openrouter.ai) and [OpenAI](https://platform.openai.com) keys.<br/>• Follow the [local-mode quickstart](./hosting/self-hosting.md) to get started. |

If you're going with a 'BYO API Key' option above (whether cloud or self-hosted), you'll need to set the `OPENROUTER_API_KEY` and `OPENAI_API_KEY` environment variables before continuing:

```bash
export OPENROUTER_API_KEY=...
export OPENAI_API_KEY=...
```

## Get Started

If you're starting on a new project, make a directory first:

```bash
mkdir your-project-dir
```

Now `cd` into your **project's directory.** 

```bash
cd your-project-dir
```

Then just give a quick the REPL help text a quick read, and you're ready go. The REPL starts in *chat mode* by default, which is good for fleshing out ideas before moving to implementation. Once the task is clear, Plandex will prompt you to switch to *tell mode* to make a detailed plan and start writing code.

```bash
plandex
```

or for short:

```bash
pdx
```

☁️ *If you're using Plandex Cloud, you'll be prompted at this point to start a trial.*

Then just give a quick the REPL help text a quick read, and you're ready go.

## REPL Flags

The REPL has a few convenient flags you can use to start it with different modes, autonomy settings, and model packs. You can pass any of these to `plandex` or `pdx` when starting the REPL.

```
  Mode 
    --chat, -c     Start in chat mode (for conversation without making changes)
    --tell, -t     Start in tell mode (for implementation)

  Autonomy 
    --no-auto      None → step-by-step, no automation
    --basic        Basic → auto-continue plans, no other automation
    --plus         Plus → auto-update context, smart context, auto-commit changes
    --semi         Semi-Auto → auto-load context
    --full         Full-Auto → auto-apply, auto-exec, auto-debug

  Models 
    --daily        Daily driver pack (default models, balanced capability, cost, and speed)
    --strong       Strong pack (more capable models, higher cost and slower)
    --cheap        Cheap pack (less capable models, lower cost and faster)
    --oss          Open source pack (open source models)
```