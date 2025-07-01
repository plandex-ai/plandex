---
sidebar_position: 4
sidebar_label: Custom Models
---

# Custom Models, Providers, and Model Packs

You can extend Plandex with custom models, providers, and model packs using a JSON file.

```bash
plandex models custom # edit the custom models file
\models custom # REPL
```

The first time you run this command, a template file with examples will be created and opened in your preferred editor.

The template file uses a JSON schema, allowing most editors to provide autocomplete, validation, and inline documentation.

## Support Levels

The level of custom model support depends on how you use Plandex.

**Self-hosted**: Everything - custom models, providers, and model packs are fully supported.

**Cloud with BYO API Keys**: Custom models and model packs. Models can only use built-in providers.

**Cloud with Integrated Models**: Custom model packs. Model packs can only use built-in models.

## Basic Example

Here's a minimal example that adds a model from Together.ai:

```json
{
  "$schema": "https://plandex.ai/schemas/models-input.schema.json",
  "providers": [
    {
      "name": "togetherai",
      "baseUrl": "https://api.together.xyz/v1",
      "apiKeyEnvVar": "TOGETHER_API_KEY"
    }
  ],
  "models": [
    {
      "modelId": "meta-llama/llama-4-maverick",
      "publisher": "meta-llama",
      "description": "Meta Llama 4 Maverick",
      "defaultMaxConvoTokens": 75000,
      "maxTokens": 1048576,
      "maxOutputTokens": 16000,
      "reservedOutputTokens": 16000,
      "preferredOutputFormat": "xml",
      "providers": [
        {
          "provider": "custom",
          "customProvider": "togetherai",
          "modelName": "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8"
        }
      ]
    }
  ]
}
```

## Custom Providers

Define providers that use OpenAI-compatible APIs:

```json
{
  "providers": [
    {
      "name": "local-llm",
      "baseUrl": "http://localhost:8080/v1",
      "skipAuth": true
    },
    {
      "name": "my-provider",
      "baseUrl": "https://api.myprovider.com/v1",
      "apiKeyEnvVar": "MY_PROVIDER_API_KEY"
    }
  ]
}
```

### Provider Settings

- `name` - Unique identifier for the provider
- `baseUrl` - API endpoint (must be OpenAI-compatible)
- `apiKeyEnvVar` - Environment variable containing the API key
- `skipAuth` - Set to `true` for local models that don't need authentication
- `extraAuthVars` - Additional authentication variables if needed

## Custom Models

Define models with their capabilities and provider mappings:

```json
{
  "$schema": "https://plandex.ai/schemas/models-input.schema.json",
  "models": [
    {
      "modelId": "my-model",
      "publisher": "My Company",
      "description": "My Custom Model",
      "defaultMaxConvoTokens": 50000,
      "maxTokens": 128000,
      "maxOutputTokens": 8192,
      "reservedOutputTokens": 8192,
      "preferredOutputFormat": "xml",
      "providers": [
        {
          "provider": "custom",
          "customProvider": "my-provider",
          "modelName": "exact-model-name-on-provider"
        },
        {
          "provider": "openrouter",
          "modelName": "my-company/my-model"
        }
      ]
    }
  ]
}
```

### Model Settings

- `modelId` - Unique identifier used in model packs
- `maxTokens` - Total token limit (input + output)
- `maxOutputTokens` - Maximum output tokens the model can generate
- `reservedOutputTokens` - Tokens reserved for output (affects effective input limit)
- `preferredOutputFormat` - Either `"xml"` or `"tool-call-json"`
- `providers` - List of providers that can serve this model

## Custom Model Packs

Create your own combinations of models for different roles:

```json
{
  "$schema": "https://plandex.ai/schemas/models-input.schema.json",
  "modelPacks": [
    {
      "name": "custom-pack",
      "description": "Custom model pack for my use case",
      "planner": "custom-model-publisher/model-name",
      "coder": "anthropic/claude-sonnet-4",
      "architect": "custom-model-publisher/model-name",
      "summarizer": "openai/gpt-4.1",
      "builder": {
        "modelId": "anthropic/claude-sonnet-4",
        "strongModel": "openai/o3-medium"
      },
      "wholeFileBuilder": {
        "modelId": "anthropic/claude-sonnet-4",
        "largeContextFallback": {
          "modelId": "google/gemini-2.5-pro",
          "largeOutputFallback": "openai/o4-mini-low"
        },
        "errorFallback": "openai/gpt-4.1"
      },
      "names": "openai/gpt-4.1-mini",
      "commitMessages": "openai/gpt-4.1-mini",
      "autoContinue": "openai/o4-mini-medium"
    }
  ]
}
```

### Model Pack Settings

- `name` - The name of the model pack
- `description` - A description of the model pack
- `localProvider` - The local provider to default to for the model pack. Currently only `ollama` is supported. This must be set for the model pack to use local models via ollama.

Custom model packs can be configured with the same [roles](./roles.md) as built-in model packs:

- `planner` (required)
- `architect` (optional, defaults to `planner`)
- `coder` (optional, defaults to `planner`)
- `summarizer` (required)
- `builder` (required)
- `wholeFileBuilder` (optional, defaults to `builder`)
- `names` (optional)
- `commitMessages` (optional)
- `autoContinue` (optional)

### Role Config

For each role set in the model pack, you can either use a simple string (the model ID) or a config object with these settings:

- `modelId` - The model to use (required when using object form)
- `temperature` - Controls randomness (0-2, role-specific defaults)
- `topP` - Alternative randomness control (0-1)
- `largeContextFallback` - Model to use when context is large
- `largeOutputFallback` - Model to use when output needs to be large
- `errorFallback` - Model to use if the primary model fails
- `strongModel` - Stronger model for complex tasks

When using a config object, all settings except `modelId` are optional.

## Complete Example

Here's a full example combining `providers`, `models`, and `modelPacks`:

```json
{
  "$schema": "https://plandex.ai/schemas/models-input.schema.json",
  "providers": [
    {
      "name": "replicate",
      "baseUrl": "https://api.replicate.com/v1",
      "apiKeyEnvVar": "REPLICATE_API_KEY"
    }
  ],
  "models": [
    {
      "modelId": "llama-4-maverick-70b",
      "publisher": "meta",
      "description": "Llama 4 Maverick 70B",
      "defaultMaxConvoTokens": 50000,
      "maxTokens": 128000,
      "maxOutputTokens": 8192,
      "reservedOutputTokens": 8192,
      "preferredOutputFormat": "xml",
      "providers": [
        {
          "provider": "custom",
          "customProvider": "replicate",
          "modelName": "meta/llama-4-maverick-70b"
        }
      ]
    }
  ],
  "modelPacks": [
    {
      "name": "hybrid-local-cloud",
      "description": "Local models for simple tasks, cloud for complex",
      "planner": "anthropic/claude-opus-4",
      "coder": "anthropic/claude-sonnet-4",
      "architect": "anthropic/claude-sonnet-4",
      "summarizer": "llama-4-maverick-70b",
      "builder": "anthropic/claude-sonnet-4",
      "names": "llama-4-maverick-70b",
      "commitMessages": "llama-4-maverick-70b",
      "autoContinue": "llama-4-maverick-70b"
    }
  ]
}
```

## Notes

- All custom providers must be OpenAI-compatible.
- Model IDs must be unique across built-in and custom models.
- Provider names must be unique across built-in and custom providers.