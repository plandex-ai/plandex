---
sidebar_position: 1
sidebar_label: Providers
---

# Model Providers

By default, Plandex uses a mix of Anthropic, OpenAI, and Google models. If an OpenAI API key is set, OpenAI models will use the OpenAI API directly. Otherwise, all models will use [OpenRouter.ai](https://openrouter.ai/).

You can also use Plandex with models from any provider that provides an OpenAI-compatible API, like the aforementioned [OpenRouter.ai](https://openrouter.ai/), [Together.ai](https://together.ai), [Replicate](https://replicate.com/), [Ollama](https://ollama.com/), and more.

## Model Performance

While you can use Plandex with many different providers and models as described above, Plandex's prompts have mainly been written and tested against the built-in models and model packs, so you should expect them to give the best results.

### Local Models Disclaimer

In particular, small models that can be run locally generally aren't strong enough to produce usable results for the [heavy-lifting roles][./roles.md] like `planner`, `architect`, `coder`, and `builder`. The prompts for these roles require very strong instruction following that is hard to achieve with small models.

The strongest open source models like DeepSeek R1 and V3 _are_ capable enough for decent results, but these models are quite large for running locally without a very powerful system. This isn't meant to discourage experimentation with local models, but to set expectations for what is realistically achievable.

## Built-in Models and Model Packs

Plandex provides a curated set of built-in models and model packs.

You can see the list of available model packs with:

```bash
\model-packs # REPL
plandex model-packs # CLI
```

You can see the list of available models with:

```bash
\models available # REPL
plandex models available # CLI
```

## Integrated Models

If you use [Plandex Cloud](../hosting/cloud.md), you have the option of using **Integrated Models Mode** which allows you to use Plandex credits to pay for AI models. No separate accounts or API keys are required in this case.

If, alternatively, you use **BYO API Key Mode** with Plandex Cloud, or if you self-host Plandex, you'll need to generate API keys for the providers you want to use.

## OpenRouter

### Account

If you don't have an OpenRouter account, first [sign up here.](https://openrouter.ai/signup)

### API Key

Once you've created an OpenRouter account, [generate an API key here.](https://openrouter.ai/keys)

## OpenAI

Note that as of v2.0.8, setting an OpenAI API key is **optional**. Plandex will use OpenRouter as a fallback for OpenAI models if no OpenAI API key is set.

If you do set an OpenAI API key, Plandex will call the OpenAI API directly for OpenAI models, which offers slightly lower latency and costs about 5% less than OpenRouter.

### Account

If you don't have an OpenAI account, first [sign up here.](https://platform.openai.com/signup)

### API Key

Once you've created an OpenAI account, [generate an API key here.](https://platform.openai.com/account/api-keys)

## Other Providers

Apart from those listed above, Plandex can use models from any provider that is compatible with the OpenAI API, like Together.ai, Replicate, Ollama, and more. You'll need to create an account and generate an API key for any other providers you plan on using.

## Environment Variables

Now that you've generated API keys for your providers, export them as environment variables in your terminal.

```bash
export OPENROUTER_API_KEY=...
export OPENAI_API_KEY=...

# optional - set api keys for any other providers you're using
export TOGETHER_API_KEY...
```

If you're using OpenAI as a model provider, you can also set a different base URL for API calls:

```bash
export OPENAI_API_BASE=... # optional - set a different base url for OpenAI calls e.g. https://<your-proxy>/v1
```

If you have multiple OpenAI orgs, you can specify which org to use:

```bash
export OPENAI_ORG_ID=... # optional - set the OpenAI OrgID if you have multiple orgs
```
