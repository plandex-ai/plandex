---
sidebar_position: 2
sidebar_label: Model Packs
---

# Built-In Model Packs

Plandex includes a curated selection of built-in model packs that have been tested and optimized for different use cases.

*A model pack is a mapping of [model roles](../models/model-roles) to [models](./built-in-models.md).*

*They can also define fallback models for large context, large output, error handling, as well as a strong variant for the `builder` role.*

## Core Packs

### `daily-driver`
*A mix of models from Anthropic, OpenAI, and Google that balances speed, quality, and cost. Supports up to 2M context.*

- **planner** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `google/gemini-2.5-pro`
    - largeContextFallback → `google/gemini-pro-1.5`
- **architect** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `google/gemini-2.5-pro`
    - largeContextFallback → `google/gemini-pro-1.5`
- **coder** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `openai/gpt-4.1`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
  - strongModel → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-medium`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-low`

### `reasoning`
*Like the daily driver, but uses sonnet-4-thinking with reasoning enabled for planning and coding. Supports up to 160k input context.*

- **planner** → `anthropic/claude-sonnet-4-thinking-hidden`
- **architect** → Uses planner model
- **coder** → `anthropic/claude-sonnet-4-thinking-hidden`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
  - strongModel → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-medium`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-low`

### `strong`
*For difficult tasks where slower responses and builds are ok. Uses o3-high for architecture and planning, claude-sonnet-4 thinking for implementation. Supports up to 160k input context.*

- **planner** → `openai/o3-high`
- **architect** → `openai/o3-high`
- **coder** → `anthropic/claude-sonnet-4-thinking-hidden`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-high`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-medium`

### `cheap`
*Cost-effective models that can still get the job done for easier tasks. Supports up to 160k context. Uses OpenAI's o4-mini model for planning, GPT-4.1 for coding, and GPT-4.1 Mini for lighter tasks.*

- **planner** → `openai/o4-mini-medium`
- **architect** → Uses planner model
- **coder** → `openai/gpt-4.1`
- **summarizer** → `openai/gpt-4.1-mini`
- **builder** → `openai/o4-mini-low`
- **wholeFileBuilder** → `openai/o4-mini-low`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-low`

### `oss`
*An experimental mix of the best open source models for coding. Supports up to 144k context, 33k per file.*

- **planner** → `deepseek/r1`
- **architect** → Uses planner model
- **coder** → `deepseek/v3`
- **summarizer** → `deepseek/r1-hidden`
- **builder** → `deepseek/r1-hidden`
- **wholeFileBuilder** → `deepseek/r1-hidden`
- **names** → `qwen/qwen3-8b-cloud`
- **commitMessages** → `qwen/qwen3-8b-cloud`
- **autoContinue** → `deepseek/r1-hidden`

## Provider Packs


### `openai`
*OpenAI blend. Supports up to 1M context. Uses OpenAI's GPT-4.1 model for heavy lifting, GPT-4.1 Mini for lighter tasks.*

- **planner** → `openai/gpt-4.1`
- **architect** → Uses planner model
- **coder** → Uses planner model
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
  - strongModel → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-medium`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-low`

### `anthropic`
*Anthropic blend. Supports up to 180k context. Uses Claude Sonnet 4 for heavy lifting, Claude 3 Haiku for lighter tasks.*

- **planner** → `anthropic/claude-sonnet-4`
- **architect** → Uses planner model
- **coder** → `anthropic/claude-sonnet-4`
- **summarizer** → `anthropic/claude-3.5-haiku`
- **builder** → `anthropic/claude-sonnet-4`
- **wholeFileBuilder** → `anthropic/claude-sonnet-4`
- **names** → `anthropic/claude-3.5-haiku`
- **commitMessages** → `anthropic/claude-3.5-haiku`
- **autoContinue** → `anthropic/claude-sonnet-4`

### `gemini-planner`
*Uses Gemini 2.5 Pro for planning, default models for other roles. Supports up to 1M input context.*

- **planner** → `google/gemini-2.5-pro`
- **architect** → Uses planner model
- **coder** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `openai/gpt-4.1`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
  - strongModel → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-medium`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-low`

### `opus-planner`
*Uses Claude Opus 4 for planning, default models for other roles. Supports up to 180k input context.*

- **planner** → `anthropic/opus-4`
- **architect** → Uses planner model
- **coder** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `openai/gpt-4.1`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
  - strongModel → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-medium`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-low`

### `o3-planner`
*Uses OpenAI o3-medium for planning, default models for other roles. Supports up to 160k input context.*

- **planner** → `openai/o3-medium`
- **architect** → Uses planner model
- **coder** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `openai/gpt-4.1`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
  - strongModel → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-medium`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-low`

### `r1-planner`
*Uses DeepSeek R1 for planning, Qwen for light tasks, and default models for implementation. Supports up to 56k input context.*

- **planner** → `deepseek/r1`
- **architect** → Uses planner model
- **coder** → `anthropic/claude-sonnet-4`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
- **wholeFileBuilder** → `openai/o4-mini-low`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-medium`

### `perplexity-planner`
*Uses Perplexity Sonar for planning, Qwen for light tasks, and default models for implementation. Supports up to 97k input context.*

- **planner** → `perplexity/sonar-reasoning`
- **architect** → Uses planner model
- **coder** → `anthropic/claude-sonnet-4`
- **summarizer** → `openai/o4-mini-low`
- **builder** → `openai/o4-mini-medium`
- **wholeFileBuilder** → `openai/o4-mini-low`
- **names** → `openai/gpt-4.1-mini`
- **commitMessages** → `openai/gpt-4.1-mini`
- **autoContinue** → `openai/o4-mini-medium`

## Local Packs

### `ollama`
*Ollama experimental local blend. Supports up to 110k context. For now, more for experimentation and benchmarking than getting work done.*

- **localProvider** → `ollama`
- **planner** → `qwen/qwen3-32b-local`
- **architect** → Uses planner model
- **coder** → Uses planner model
- **summarizer** → `mistral/devstral-small`
- **builder** → `mistral/devstral-small`
- **wholeFileBuilder** → `mistral/devstral-small`
- **names** → `qwen/qwen3-8b-local`
- **commitMessages** → `qwen/qwen3-8b-local`
- **autoContinue** → `mistral/devstral-small`

### `ollama-daily`
*Ollama adaptive/daily-driver blend. Uses 'daily-driver' for heavy lifting, local models for lighter tasks.*

- **localProvider** → `ollama`
- **planner** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `google/gemini-2.5-pro`
    - largeContextFallback → `google/gemini-pro-1.5`
- **architect** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `google/gemini-2.5-pro`
    - largeContextFallback → `google/gemini-pro-1.5`
- **coder** → `anthropic/claude-sonnet-4`
  - largeContextFallback → `openai/gpt-4.1`
- **summarizer** → `mistral/devstral-small`
- **builder** → `openai/o4-mini-medium`
  - strongModel → `openai/o4-mini-high`
- **wholeFileBuilder** → `openai/o4-mini-medium`
- **names** → `qwen/qwen3-8b-local`
- **commitMessages** → `qwen/qwen3-8b-local`
- **autoContinue** → `openai/o4-mini-low`

### `ollama-oss`
*Ollama adaptive/oss blend. Uses local models for planning and context selection, open source cloud models for implementation and file edits. Supports up to 110k context.*

- **localProvider** → `ollama`
- **planner** → `deepseek/r1`
- **architect** → Uses planner model
- **coder** → `deepseek/v3`
- **summarizer** → `mistral/devstral-small`
- **builder** → `deepseek/r1-hidden`
- **wholeFileBuilder** → `deepseek/r1-hidden`
- **names** → `qwen/qwen3-8b-local`
- **commitMessages** → `qwen/qwen3-8b-local`
- **autoContinue** → `deepseek/r1-hidden`
