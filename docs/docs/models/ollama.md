---
sidebar_position: 6
sidebar_label: Ollama Quickstart
---

# Ollama Quickstart

Plandex works with [Ollama](https://ollama.com/) models. To use them, you need to [self-host Plandex.](../hosting/self-hosting/local-mode-quickstart.md) Ollama isn't supported with Plandex Cloud.

## Disclaimer

While local models are supported via Ollama, small models that can be run locally often aren't strong enough to produce usable results for the [heavy-lifting roles](./roles.md) like `planner`, `architect`, `coder`, and `builder`. The prompts for these roles require strong instruction following that can be hard to achieve with small models.

The strongest open source models _are_ capable enough for decent results, but these models are quite large for running locally without a very powerful system. This isn't meant to discourage experimentation with local models, but to set expectations for what is achievable.

To help bridge the gap as local models continue to improve their capabilities, a built-in `ollama-adaptive` model pack is available. This model pack uses local Ollama models for less demanding roles, plus larger remote models for heavy-lifting. There's also a built-in `ollama-experimental` model pack that uses local models for all roles—this is recommended for testing and benchmarking, but not for getting real work done.

## Install and run Ollama

[Download and install ollama](https://ollama.com/download) for your platform.

Then make sure the ollama server is running:

```bash
ollama serve
```

## Pull Ollama models

Pull the models you want to use. For the built-in `ollama-adaptive` and `ollama-experimental` model packs, pull the following models:

```bash
ollama pull qwen3:32b
ollama pull qwen3:8b
ollama pull qwen3:14b
ollama pull devstral:24b
```

## Use Ollama in Plandex

### Built-in model packs

To use one of the built-in Ollama model packs in Plandex, decide whether you want to use `ollama-experimental`, which uses local models for all roles, but may struggle in practice, or `ollama-adaptive`, which uses local models for less demanding roles, plus the default Plandex models for heavy-lifting.

```bash
\set-model ollama-experimental # REPL
plandex set-model ollama-experimental # CLI
```

Or:

```bash
\set-model ollama-adaptive # REPL
plandex set-model ollama-adaptive # CLI
```

### Custom models and model packs

You can also setup [custom models and model packs](./custom-models.md) for use with Ollama.

When configuring a custom model, be sure you add the `ollama` provider to the `providers` array with the `modelName` set to the name of the model you want to use, exactly as it appears in the [Ollama model list](https://ollama.com/models), prefixed with `ollama_chat/`. For example, to use the `qwen3:32b` model, you would add the following to the `providers` array:

```json
"providers": [
  {
    "provider": "ollama",
    "modelName": "ollama_chat/qwen3:32b"
  }
]
```

When configuring a custom model pack to use Ollama, set the top-level `localProvider` key to `ollama`. For example:

```json
{
  ...
  "modelPacks": [
    {
      "localProvider": "ollama",
      ...
    }
  ]
}
```

