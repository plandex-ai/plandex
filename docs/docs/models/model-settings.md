---
sidebar_position: 3
sidebar_label: Settings
---

# Model Settings

Plandex gives you a number of ways to control the models used in your plans. Changes to models are [version controlled](../core-concepts/version-control.md) and can be [branched](../core-concepts/branches.md).

## `models` and `set-model`

You can see the current plan's models with the `models` command and change them with the `set-model` command.

```bash
plandex models # show the current models
plandex set-model # select a model pack or configure model settings in JSON
```

## Model Defaults 

`set-model` updates model settings for the current plan. If you want to change the default model settings for all new plans, use `set-model default`.

```bash
plandex models default # show the default models for new plans
plandex set-model default # select a default model pack for new plans or configure default model settings in JSON
```

## Model Settings JSON

If you select the 'edit JSON' option in either the `set-model` or `set-model default` commands, or you use the `--json` flag, you can edit the model settings in a JSON file in your preferred editor.

The models file lets you configure which model to use for each role, along with settings like temperature/top-p and fallback models. It uses a JSON schema, allowing most editors to provide autocomplete, validation, and inline documentation.

Model roles can either be a string (the model ID) or an object with model config.

### Basic Example

```json
{
  "$schema": "https://plandex.ai/schemas/model-pack-inline.schema.json",
  "planner": "anthropic/claude-opus-4",
  "coder": "anthropic/claude-sonnet-4",
  "architect": "anthropic/claude-sonnet-4",
  "summarizer": "anthropic/claude-3.5-haiku",
  "builder": "anthropic/claude-sonnet-4",
  "names": "anthropic/claude-3.5-haiku",
  "commitMessages": "anthropic/claude-3.5-haiku",
  "autoContinue": "anthropic/claude-3.5-haiku"
}
```

### Advanced Example

You can also configure individual role settings and fallbacks with an object:

```json
{
  "$schema": "https://plandex.ai/schemas/model-pack-inline.schema.json",
  "planner": {
    "modelId": "anthropic/claude-opus-4",
    "temperature": 0.7,
    "topP": 0.9,
    "largeContextFallback": "google/gemini-2.5-pro"
  },
  "coder": "anthropic/claude-sonnet-4",
  "architect": "anthropic/claude-sonnet-4",
  "summarizer": "anthropic/claude-3.5-haiku",
  "builder": {
    "modelId": "anthropic/claude-sonnet-4",
    "errorFallback": "openai/gpt-4.1"
  },
  "wholeFileBuilder": {
    "modelId": "anthropic/claude-sonnet-4",
    "largeContextFallback": {
      "modelId": "google/gemini-2.5-pro",
      "largeOutputFallback": "openai/o4-mini-low"
    }
  },
  "names": "anthropic/claude-3.5-haiku",
  "commitMessages": "anthropic/claude-3.5-haiku",
  "autoContinue": "anthropic/claude-3.5-haiku"
}
```

### Role Config

For each role, you can either use a simple string (the model ID) or a config object with these settings:

- `modelId` - The model to use (required when using object form)
- `temperature` - Controls randomness (0-2, role-specific defaults)
- `topP` - Alternative randomness control (0-1)
- `largeContextFallback` - Model to use when context is large
- `largeOutputFallback` - Model to use when output needs to be large
- `errorFallback` - Model to use if the primary model fails
- `strongModel` - Stronger model for complex tasks

When using a config object, all settings except `modelId` are optional.

## Local Provider

You can set the top-level `localProvider` key to `ollama` to use local models via [Ollama](https://ollama.com/):

```json
{
  "$schema": "https://plandex.ai/schemas/model-pack-inline.schema.json",
  "localProvider": "ollama",
  "planner": "ollama/deepseek-r1:14b",
  ...
}
```
