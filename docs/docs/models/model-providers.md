---
sidebar_position: 2
sidebar_label: Providers
---

# Model Providers

If you use [Plandex Cloud](../hosting/cloud.md) in **Integrated Models Mode**, you can use Plandex credits to pay for AI models. No separate accounts or API keys are required in this case.

If you instead use **BYO API Key Mode** with Plandex Cloud, or if you self-host Plandex, you'll need to set API keys (or other credentials) for the providers you want to use.

To see available providers, run:

```bash
\providers # REPL
plandex providers # CLI
```

## API Keys / Environment Variables

API keys or credentials are set through **environment variables** when running the Plandex CLI. For example:

```bash
export OPENROUTER_API_KEY=...
export OPENAI_API_KEY=...
export ANTHROPIC_API_KEY=...

plandex # start the Plandex REPL
```

## Provider Selection

Many models can be served by multiple different providers. For example, OpenAI models are available via OpenAI, Microsoft Azure, and OpenRouter.

When multiple providers are available for a model, which provider is selected depends on the authentication environment variables that are set when running the CLI or REPL. If environment variables are set for multiple providers, the direct provider takes precedence. For example, if you set both `ANTHROPIC_API_KEY` (for the direct Anthropic API) and `OPENROUTER_API_KEY` (for OpenRouter), Plandex will use the direct Anthropic API for Anthropic models.

## Built-In Providers

### OpenRouter

Apart from Plandex Cloud's Integrated Models Mode, the quickest way to get started is to use [OpenRouter.ai](https://openrouter.ai/), which allows you to use a wide range of models—including all those Plandex uses by default—with a single account and API key.

To use OpenRouter, you'll need to create an account and generate an API key, then set the `OPENROUTER_API_KEY` environment variable.

```bash
export OPENROUTER_API_KEY=...

plandex # start the Plandex REPL
```

You can also use OpenRouter alongside other providers. For example, if you set both `OPENROUTER_API_KEY` and `OPENAI_API_KEY`, Plandex will use the OpenAI API directly for OpenAI models and OpenRouter for other models. Using direct providers offers slightly lower latency and costs about 5% less than OpenRouter.

If you set a `OPENROUTER_API_KEY` and are also using other providers, Plandex will also **fail over** to OpenRouter if another provider has an error. This offers a strong layer of redundancy since OpenRouter itself routes model calls across a number of different upstream providers.

### OpenAI

You can optionally set an `OPENAI_API_KEY` to use the OpenAI API directly with your own OpenAI account when calling OpenAI models.

```bash
export OPENAI_API_KEY=... # set your OpenAI API key for OpenAI models
export OPENAI_ORG_ID=... # optionally set your OpenAI OrgID if you have multiple orgs
```

### Anthropic

You can optionally set an `ANTHROPIC_API_KEY` to use the Anthropic API directly with your own Anthropic account when calling Anthropic models.

```bash
export ANTHROPIC_API_KEY=... # set your Anthropic API key for Anthropic models
```

### Google AI Studio

You can optionally set a `GEMINI_API_KEY` to use Google Gemini models with your own Google account via Google AI Studio.

```bash
export GEMINI_API_KEY=... # set your Google AI Studio API key for Google Gemini models
```

### Google Vertex AI

You can optionally use Google Vertex AI, which offers models from Gemini, Anthropic, and more. Vertex authentication requires a few environment variables to be set.

```bash
export GOOGLE_APPLICATION_CREDENTIALS=... # either a path to a JSON file, the JSON itself as a string, or the base64 encoded JSON
export VERTEXAI_PROJECT=... # your Vertex project ID
export VERTEXAI_LOCATION=... # your Vertex location (e.g. us-east5)
```

### Azure

You can optionally use Microsoft Azure for OpenAI models. Azure authentication requires both an API key and a base URL.

```bash
export AZURE_OPENAI_API_KEY=... # set your Azure OpenAI API key
export AZURE_API_BASE=... # set your Azure API base URL (required)
export AZURE_API_VERSION=... # optionally set your Azure API version - defaults to 2025-04-01-preview
export AZURE_DEPLOYMENTS_MAP='{"gpt-4.1": "gpt-4.1-deployment-name"}' # optionally set a map of model names to deployment names with a JSON object (only needed if deployment names are different from model names)
```

### AWS Bedrock

You can optionally use AWS Bedrock for Anthropic models. AWS Bedrock uses standard AWS authentication via environment variables.

```bash
export AWS_ACCESS_KEY_ID=... # set your AWS access key ID
export AWS_SECRET_ACCESS_KEY=... # set your AWS secret access key
export AWS_REGION=... # set your AWS region (e.g. us-east-1)

export AWS_SESSION_TOKEN=... # optionally set your AWS session token
export AWS_INFERENCE_PROFILE_ARN=... # optionally set your AWS inference profile ARN
```

### DeepSeek

You can optionally use DeepSeek models with your own DeepSeek account.

```bash
export DEEPSEEK_API_KEY=... # set your DeepSeek API key for DeepSeek models
```

### Perplexity

You can optionally use Perplexity models with your own Perplexity account.

```bash
export PERPLEXITY_API_KEY=... # set your Perplexity API key for Perplexity models
```

## Custom Providers

If you're self-hosting Plandex, you can also use with models from any provider that provides an OpenAI-compatible API.

To configure custom providers, you can use a dev-friendly JSON config file:

```bash
\models custom # REPL
plandex models custom # CLI
```

[More details on custom providers](./custom-models.md)

## Local Models

Plandex supports local models via [Ollama](https://ollama.com/). It doesn't require any authentication or API keys.
 
For more details, see the [Ollama Quickstart](./ollama.md).