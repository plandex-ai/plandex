---
sidebar_position: 1
sidebar_label: Providers
---

# Model Providers

By default, Plandex uses OpenAI models, but you can use models from any provider that provides an OpenAI-compatible API, like [OpenRouter.ai](https://openrouter.ai/) (Anthropic, Gemini, and open source models), [Together.ai](https://together.ai) (open source models), [Replicate](https://replicate.com/), [Ollama](https://ollama.com/), and more.

## Limitations

While you can use Plandex with many different providers and models as described above, Plandex requires reliable function calling, which can still be a challenge to find in non-OpenAI models. Additionally, Plandex's prompts have mainly been written and tested against OpenAI models.  

For these reasons, OpenAI models will tend to provide the best experience, and it's recommended to start with the defaults.

In the future, we plan to offer prompts that are tailored for different models, and we also expect that other models and providers will catch up on the reliability front. Until then, **support for non-OpenAI models should generally be considered experimental.**

## OpenAI

### Account

If you don't have an OpenAI account, first [sign up here.](https://platform.openai.com/signup)

### API Key

Once you've created an OpenAI account, [generate an API key here.](https://platform.openai.com/account/api-keys)

## Other Providers

Plandex can use models from any provider that is compatible with the OpenAI API, like OpenRouter.ai (Anthropic, Gemini, and open source models), Together.ai (open source models), Replicate, Ollama, and more. You'll need to create an account and generate an API key for any other providers you plan on using.

## Environment Variables

Now that you've generated an API key, export it as an environment variable in your terminal.

```bash
export OPENAI_API_KEY=...

# optional - set api keys for any other providers you're using
export OPENROUTER_API_KEY=...
export TOGETHER_API_KEY...
```

If you're using OpenAI as your model provider, you can also set a different base URL for API calls:

```bash
export OPENAI_API_BASE=... # optional - set a different base url for OpenAI calls e.g. https://<your-proxy>/v1
```

If you have multiple OpenAI orgs, you can specify which org to use:

```bash
export OPENAI_ORG_ID=... # optional - set the OpenAI OrgID if you have multiple orgs
```