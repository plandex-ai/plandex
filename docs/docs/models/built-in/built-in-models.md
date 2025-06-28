---
sidebar_position: 1
sidebar_label: Models
---

# Built-In Models

Plandex includes a curated selection of built-in models.

## OpenAI

### `openai/o3-high`

- OpenAI o3 (high reasoning)
- Max Tokens: 200k
- Max Output: 100k
- Reserved Output: 40k
- Effective Input: 160k
- Features: XML output, no system prompt, fixed parameters, reasoning effort
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/o3-medium`

- OpenAI o3 (medium reasoning)
- Max Tokens: 200k
- Max Output: 100k
- Reserved Output: 40k
- Effective Input: 160k
- Features: XML output, no system prompt, fixed parameters, reasoning effort
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/o3-low`

- OpenAI o3 (low reasoning)
- Max Tokens: 200k
- Max Output: 100k
- Reserved Output: 40k
- Effective Input: 160k
- Features: XML output, no system prompt, fixed parameters, reasoning effort
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/o4-mini-high`

- OpenAI o4-mini (high reasoning)
- Max Tokens: 200k
- Max Output: 100k
- Reserved Output: 40k
- Effective Input: 160k
- Features: JSON output, no system prompt, fixed parameters, reasoning effort
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/o4-mini-medium`

- OpenAI o4-mini (medium reasoning)
- Max Tokens: 200k
- Max Output: 100k
- Reserved Output: 30k
- Effective Input: 170k
- Features: JSON output, no system prompt, fixed parameters, reasoning effort
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/o4-mini-low`

- OpenAI o4-mini (low reasoning)
- Max Tokens: 200k
- Max Output: 100k
- Reserved Output: 20k
- Effective Input: 180k
- Features: JSON output, no system prompt, fixed parameters, reasoning effort
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/gpt-4.1`

- OpenAI GPT-4.1
- Max Tokens: 1,047,576
- Max Output: 32,768
- Reserved Output: 32,768
- Effective Input: 1,014,808
- Features: JSON output, full compatibility
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/gpt-4.1-mini`

- OpenAI GPT-4.1 Mini
- Max Tokens: 1,047,576
- Max Output: 32,768
- Reserved Output: 32,768
- Effective Input: 1,014,808
- Features: JSON output, full compatibility
- Providers: OpenAI, Azure OpenAI, OpenRouter

### `openai/gpt-4.1-nano`

- OpenAI GPT-4.1 Nano
- Max Tokens: 1,047,576
- Max Output: 32,768
- Reserved Output: 32,768
- Effective Input: 1,014,808
- Features: JSON output, full compatibility
- Providers: OpenAI, Azure OpenAI, OpenRouter

## Anthropic

### `anthropic/claude-opus-4`

- Anthropic Claude Opus 4
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 20k
- Effective Input: 180k
- Features: XML output, cache control, single message mode
- Providers: Anthropic, AWS Bedrock, Google Vertex, OpenRouter

### `anthropic/claude-sonnet-4`

- Anthropic Claude Sonnet 4
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 40k
- Effective Input: 160k
- Features: XML output, cache control, single message mode
- Providers: Anthropic, AWS Bedrock, Google Vertex, OpenRouter

### `anthropic/claude-sonnet-4-thinking`

- Claude Sonnet 4 (visible reasoning)
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 40k
- Effective Input: 160k
- Features: XML output, cache control, single message mode, reasoning budget 32k
- Providers: Anthropic, AWS Bedrock, Google Vertex, OpenRouter

### `anthropic/claude-sonnet-4-thinking-hidden`

- Claude Sonnet 4 (hidden reasoning)
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 40k
- Effective Input: 160k
- Features: XML output, cache control, single message mode, reasoning budget 32k
- Providers: Anthropic, AWS Bedrock, Google Vertex, OpenRouter

### `anthropic/claude-3.7-sonnet`

- Anthropic Claude 3.7 Sonnet
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 20k
- Effective Input: 180k
- Features: XML output, cache control, single message mode
- Providers: Anthropic, AWS Bedrock, Google Vertex, OpenRouter

### `anthropic/claude-3.7-sonnet-thinking`

- Claude 3.7 Sonnet (visible reasoning)
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 20k
- Effective Input: 180k
- Features: XML output, cache control, single message mode, reasoning budget 32k
- Providers: Anthropic, AWS Bedrock, Google Vertex, OpenRouter

### `anthropic/claude-3.7-sonnet-thinking-hidden`

- Claude 3.7 Sonnet (hidden reasoning)
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 20k
- Effective Input: 180k
- Features: XML output, cache control, single message mode, reasoning budget 32k
- Providers: Anthropic, AWS Bedrock, Google Vertex, OpenRouter

### `anthropic/claude-3.5-sonnet`

- Anthropic Claude 3.5 Sonnet
- Max Tokens: 200k
- Max Output: 128k
- Reserved Output: 20k
- Effective Input: 180k
- Features: XML output, cache control, single message mode
- Providers: Anthropic, Google Vertex, AWS Bedrock, OpenRouter

### `anthropic/claude-3.5-haiku`

- Anthropic Claude 3.5 Haiku
- Max Tokens: 200k
- Max Output: 8,192
- Reserved Output: 8,192
- Effective Input: 191,808
- Features: XML output, cache control, single message mode
- Providers: Anthropic, Google Vertex, AWS Bedrock, OpenRouter

## Google

### `google/gemini-2.5-pro`

- Google Gemini 2.5 Pro
- Max Tokens: 1,048,576
- Max Output: 65,535
- Reserved Output: 65,535
- Effective Input: 983,041
- Features: XML output
- Providers: Google AI Studio, Google Vertex, OpenRouter

### `google/gemini-2.5-flash`

- Google Gemini 2.5 Flash
- Max Tokens: 1,048,576
- Max Output: 65,535
- Reserved Output: 65,535
- Effective Input: 983,041
- Features: XML output
- Providers: Google AI Studio, Google Vertex, OpenRouter

### `google/gemini-2.5-flash-thinking`

- Gemini 2.5 Flash (visible reasoning)
- Max Tokens: 1,048,576
- Max Output: 65,535
- Reserved Output: 65,535
- Effective Input: 983,041
- Features: XML output, visible reasoning
- Providers: Google AI Studio, Google Vertex, OpenRouter

### `google/gemini-2.5-flash-thinking-hidden`

- Gemini 2.5 Flash (hidden reasoning)
- Max Tokens: 1,048,576
- Max Output: 65,535
- Reserved Output: 65,535
- Effective Input: 983,041
- Features: XML output, hidden reasoning
- Providers: Google AI Studio, Google Vertex, OpenRouter

### `google/gemini-pro-1.5`

- Google Gemini 1.5 Pro
- Max Tokens: 2M
- Max Output: 8,192
- Reserved Output: 8,192
- Effective Input: 1,991,808
- Features: XML output
- Providers: Google AI Studio, Google Vertex, OpenRouter

## DeepSeek

### `deepseek/v3`

- DeepSeek V3
- Max Tokens: 64k
- Max Output: 8,192
- Reserved Output: 8,192
- Effective Input: 55,808
- Features: XML output
- Providers: DeepSeek, OpenRouter

### `deepseek/r1`

- DeepSeek R1 (visible reasoning)
- Max Tokens: 164k
- Max Output: 33k
- Reserved Output: 20k
- Effective Input: 144k
- Features: XML output, visible reasoning
- Providers: DeepSeek, OpenRouter

### `deepseek/r1-hidden`

- DeepSeek R1 (hidden reasoning)
- Max Tokens: 164k
- Max Output: 33k
- Reserved Output: 20k
- Effective Input: 144k
- Features: XML output, hidden reasoning
- Providers: DeepSeek, OpenRouter

### `deepseek/r1-70b`

- DeepSeek R1 70B (Ollama only)
- Max Tokens: 131,072
- Max Output: 131,072
- Reserved Output: 20k
- Effective Input: 111,072
- Features: XML output
- Providers: Ollama

### `deepseek/r1-32b`

- DeepSeek R1 32B (Ollama only)
- Max Tokens: 131,072
- Max Output: 131,072
- Reserved Output: 20k
- Effective Input: 111,072
- Features: XML output
- Providers: Ollama

### `deepseek/r1-14b`

- DeepSeek R1 14B (Ollama only)
- Max Tokens: 131,072
- Max Output: 131,072
- Reserved Output: 20k
- Effective Input: 111,072
- Features: XML output
- Providers: Ollama

### `deepseek/r1-8b`

- DeepSeek R1 8B (Ollama only)
- Max Tokens: 131,072
- Max Output: 131,072
- Reserved Output: 20k
- Effective Input: 111,072
- Features: XML output
- Providers: Ollama

## Perplexity

### `perplexity/r1-1776`

- Perplexity R1-1776 (visible reasoning)
- Max Tokens: 128k
- Max Output: 128k
- Reserved Output: 30k
- Effective Input: 98k
- Features: XML output, visible reasoning
- Providers: Perplexity, OpenRouter

### `perplexity/r1-1776-hidden`

- Perplexity R1-1776 (hidden reasoning)
- Max Tokens: 128k
- Max Output: 128k
- Reserved Output: 30k
- Effective Input: 98k
- Features: XML output, hidden reasoning
- Providers: Perplexity, OpenRouter

### `perplexity/r1-1776-70b`

- Perplexity R1-1776 70B (Ollama only)
- Max Tokens: 131,072
- Max Output: 131,072
- Reserved Output: 20k
- Effective Input: 111,072
- Features: XML output
- Providers: Ollama

### `perplexity/sonar-reasoning`

- Perplexity Sonar Reasoning (visible reasoning)
- Max Tokens: 127k
- Max Output: 127k
- Reserved Output: 30k
- Effective Input: 97k
- Features: XML output, visible reasoning
- Providers: Perplexity, OpenRouter

### `perplexity/sonar-reasoning-hidden`

- Perplexity Sonar Reasoning (hidden reasoning)
- Max Tokens: 127k
- Max Output: 127k
- Reserved Output: 30k
- Effective Input: 97k
- Features: XML output, hidden reasoning
- Providers: Perplexity, OpenRouter

## Qwen

### `qwen/qwen-2.5-coder-32b-instruct`

- Qwen 2.5 Coder 32B
- Max Tokens: 128k
- Max Output: 8,192
- Reserved Output: 8,192
- Effective Input: 119,808
- Features: XML output
- Providers: OpenRouter

### `qwen/qwen3-235b-local`

- Qwen 3-235B (Ollama only)
- Max Tokens: 40,960
- Max Output: 40,960
- Reserved Output: 8,192
- Effective Input: 32,768
- Features: XML output
- Providers: Ollama

### `qwen/qwen3-235b-a22b-cloud`

- Qwen 3-235B A22B
- Max Tokens: 40,960
- Max Output: 40,960
- Reserved Output: 8,192
- Effective Input: 32,768
- Features: XML output
- Providers: OpenRouter

### `qwen/qwen3-32b-local`

- Qwen 3-32B (Ollama only)
- Max Tokens: 40,960
- Max Output: 40,960
- Reserved Output: 8,192
- Effective Input: 32,768
- Features: XML output
- Providers: Ollama

### `qwen/qwen3-32b-cloud`

- Qwen 3-32B
- Max Tokens: 40,960
- Max Output: 40,960
- Reserved Output: 8,192
- Effective Input: 32,768
- Features: XML output
- Providers: OpenRouter

### `qwen/qwen3-14b-local`

- Qwen 3-14B (Ollama only)
- Max Tokens: 40,960
- Max Output: 40,960
- Reserved Output: 8,192
- Effective Input: 32,768
- Features: XML output
- Providers: Ollama

### `qwen/qwen3-14b-cloud`

- Qwen 3-14B
- Max Tokens: 40,960
- Max Output: 40,960
- Reserved Output: 8,192
- Effective Input: 32,768
- Features: XML output
- Providers: OpenRouter

### `qwen/qwen3-8b-local`

- Qwen 3-8B (Ollama only)
- Max Tokens: 32,768
- Max Output: 32,768
- Reserved Output: 8,192
- Effective Input: 24,576
- Features: XML output
- Providers: Ollama

### `qwen/qwen3-8b-cloud`

- Qwen 3-8B
- Max Tokens: 128k
- Max Output: 20k
- Reserved Output: 20k
- Effective Input: 108k
- Features: XML output
- Providers: OpenRouter

## Mistral

### `mistral/devstral-small`

- Mistral Devstral Small
- Max Tokens: 128k
- Max Output: 128k
- Reserved Output: 16,384
- Effective Input: 111,616
- Features: XML output
- Providers: Ollama, OpenRouter