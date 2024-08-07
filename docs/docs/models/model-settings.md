---
sidebar_position: 3
sidebar_label: Settings
---

# Model Settings

Plandex gives you a number of ways to control the models and models settings used in your plans. Changes to models and model settings are [version controlled](../core-concepts/version-control.md) and can be [branched](../core-concepts/branches.md).

## `models` and `set-model`

You can see the current plan's models and model settings with the `models` command and change them with the `set-model` command.

```bash
plandex models # show the current AI models and model settings
plandex models available # show all available models
plandex set-model # select from a list of models and settings
plandex set-model planner openrouter/anthropic/claude-3.5-sonnet # set the main planner model to Claude Sonnet 3.5 from OpenRouter.ai
plandex set-model builder temperature 0.1 # set the builder model's temperature to 0.1
plandex set-model max-tokens 4000 # set the planner model overall token limit to 4000
plandex set-model max-convo-tokens 20000  # set how large the conversation can grow before Plandex starts using summaries
```

## Model Defaults  

`set-model` updates model settings for the current plan. If you want to change the default model settings for all new plans, use `set-model default`.

```bash
plandex models default # show the default model settings
plandex set-model default # select from a list of models and settings
plandex set-model default planner openai/gpt-4 # set the default planner model to OpenAI gpt-4
```

## Custom Models

Use `models add` to add a custom model and use any provider that is compatible with OpenAI, including OpenRouter.ai, Together.ai, Ollama, Replicate, and more.

```bash
plandex models add # add a custom model
plandex models available --custom # show all available custom models
plandex models delete # delete a custom model
```

## Model Packs

Instead of changing models for each role one by one, a model pack lets you switch out all roles at once. You can create your own model packs with `model-packs create`, list built-in and custom model packs with `model-packs`, and remove custom model packs with `model-packs delete`.

```bash
plandex set-model # select from a list of model packs for the current plan
plandex set-model default # select from a list of model packs to set as the default for all new plans
plandex set-model anthropic-claude-3.5-sonnet-gpt-4o # set the current plan's model pack by name
plandex set-model default Mixtral-8x22b/Mixtral-8x7b/gpt-4o # set the default model pack for all new plans

plandex model-packs # list built-in and custom model packs
plandex model-packs create # create a new custom model pack
plandex model-packs --custom # list only custom model packs
```