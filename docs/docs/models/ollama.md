---
sidebar_position: 7
sidebar_label: Ollama Quickstart
---

# Ollama Quickstart

Plandex works with [Ollama](https://ollama.com/) models. To use them, you need to [self-host Plandex.](../hosting/self-hosting/local-mode-quickstart.md)
**Ollama isn't supported with Plandex Cloud.**

## Disclaimer

While local models are supported via Ollama, small models that can be run locally often aren't strong enough to produce usable results for the [heavy-lifting roles](./roles.md) like `planner`, `architect`, `coder`, and `builder`. The prompts for these roles require strong instruction following that can be hard to achieve with small models.

The strongest open source models _are_ capable enough for decent results, but these models are quite large for running locally without a very powerful system. This isn't meant to discourage experimentation with local models, but to set expectations for what is achievable.

To help bridge the gap as local models continue to improve their capabilities, two 'adaptive' model packs are available: `ollama-oss` and `ollama-daily`. These model packs use local Ollama models for less demanding roles, plus larger remote models for heavy-lifting (open source models for the `oss` variant, and the same models used in the default `daily-driver` model pack for the `daily` variant).

Over time, as local models improve, these adaptive model packs will be updated to use local models for more roles.

There's also a built-in experimental `ollama` model pack that uses local models for all roles—this is recommended for testing and benchmarking, but not (yet) for getting real work done.

## System requirements

To use Ollama models, you need enough system resources to run the models you want to use. To use the built-in experimental `ollama` model pack (with qwen3:32b as the largest model), at least 32GB of RAM is recommended as an absolute minimum—48GB or more is recommended for breathing room. For the `ollama-daily` and `ollama-oss` model packs (with devstral:24b as the largest model), at least 16GB of RAM is recommended as an absolute minimum—24GB or more is recommended for breathing room.

If you use [custom models and a custom model pack](./custom-models.md), you'll have full flexibility to choose the appropriate models for your system. Just remember that running Plandex prompts successfully is a challenge for even the largest local models.

## Install and run Ollama

[Download and install ollama](https://ollama.com/download) for your platform.

Then make sure the Ollama server is running:

```bash
ollama serve
```

## Pull Ollama models

Pull the models you want to use. 

For the built-in `ollama` model pack, pull the following models:

```bash
ollama pull qwen3:8b
ollama pull qwen3:14b
ollama pull qwen3:32b
ollama pull devstral:24b
```

For the `ollama-daily` and `ollama-oss` model packs, pull the following models:

```bash
ollama pull qwen3:8b
ollama pull devstral:24b
```

## Use Ollama in Plandex

### Built-in model packs

To use one of the built-in Ollama model packs in Plandex, decide whether you want to use `ollama`, which uses local models for all roles, but may struggle in practice, `ollama-daily`, which uses local models for less demanding roles, plus the default Plandex models from the `daily-driver` model pack for heavy-lifting, or `ollama-oss`, which uses local models for less demanding roles, plus the open source Plandex models from the `oss` model pack for heavy-lifting.

```bash
\set-model ollama # REPL
plandex set-model ollama # CLI
```

Or:

```bash
\set-model ollama-daily # REPL
\set-model ollama-oss
plandex set-model ollama-daily # CLI
plandex set-model ollama-oss
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

## Contributors

If you experiment with Plandex and Ollama and you find model combinations that work better than the built-in model packs, please chime in on [Discord](https://discord.gg/plandex-ai) or [open a PR](https://github.com/plandex-ai/plandex/pulls). The world of local models moves fast, and we can't always keep up with the cutting edge ourselves, so it would be great to have community help on filling in the gaps. With your help, we'd love to get to a place where Plandex can work effectively with 100% local models.