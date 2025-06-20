---
sidebar_position: 1
sidebar_label: Overview
---

# Models Overview

By default, Plandex uses a mix of Anthropic, OpenAI, and Google models. While this is a good starting point, and is the recommended way to use Plandex for most users, full customization of models and providers is also supported.

## Roles and Model Packs

Plandex has multiple [roles](./roles.md) which are responsible for different aspects of planning, coding, and applying changes. Each role can be assigned a different model. A **model pack** is a mapping of roles to specific models.

## Built-in Models and Model Packs

Plandex provides a curated set of built-in models and model packs.

You can see the list of available model packs with:

```bash
\model-packs # REPL
plandex model-packs # CLI
```

You can see the details of a specific model pack with:

```bash
\model-packs show model-pack-name # REPL
plandex model-packs show model-pack-name # CLI
```

You can see the list of available models with:

```bash
\models available # REPL
plandex models available # CLI
```

## Model Settings

You can see the model settings for the current plan with:

```bash
\models # REPL
plandex models # CLI
```

And you can see the default model settings for new plans with:

```bash
\models default # REPL
plandex models default # CLI
```

You can change the model settings for the current plan with:

```bash
\set-model # REPL
plandex set-model # CLI
```

And you can set the default model settings for new plans with:

```bash
\set-model default # REPL
plandex set-model default # CLI
```

[More details on model settings](./model-settings.md)

## Providers

Plandex offers flexibility on the providers you can use to serve models.

If you use [Plandex Cloud](../hosting/cloud.md) in **Integrated Models Mode**, you can use Plandex credits to pay for AI models. In that case, you won't need to worry about providers, provider accounts, or API keys.

If instead you use **BYO API Key Mode** with Plandex Cloud, or if you [self-host](../hosting/self-hosting/local-mode-quickstart.md) Plandex, you'll need to set API keys (or other credentials) for the providers you want to use. Multiple built-in providers are supported. 

If you're self-hosting, you can also configure custom providers—they just need to be OpenAI-compatible.

[More details on providers](./model-providers.md)

## Custom Models, Providers, and Model Packs

You can configure custom models, providers, and model packs with a dev-friendly JSON config file:

```bash
\models custom # REPL
plandex models custom # CLI
```

[More details on custom models, providers, and model packs](./custom-models.md)

## Model Performance

While you can use Plandex with many different providers and models as described above, Plandex's prompts have mainly been written and tested against the built-in models and model packs, so you should expect them to give the best results.

## Local Models

Plandex supports local models via [Ollama](https://ollama.com/). For more details, see the [Ollama Quickstart](./ollama.md).

### Local Models Disclaimer

While local models are supported via Ollama, small models that can be run locally often aren't strong enough to produce usable results for the [heavy-lifting roles](./roles.md) like `planner`, `architect`, `coder`, and `builder`. The prompts for these roles require strong instruction following that can be hard to achieve with small models.

The strongest open source models _are_ capable enough for decent results, but these models are quite large for running locally without a very powerful system. This isn't meant to discourage experimentation with local models, but to set expectations for what is realistically achievable.

To help bridge the gap as local models continue to improve their capabilities, a built-in `ollama-adaptive` model pack is available. This model pack uses local Ollama models for less demanding roles, plus larger remote models for heavy-lifting.  



