---
sidebar_position: 3
sidebar_label: Claude Pro/Max
---

# Claude Pro/Max Subscription

If you have a Claude Pro or Max subscription, Plandex can use it when calling Anthropic models. You can use it in either Integrated Models Mode on Plandex Cloud, or in BYO Key Mode (whether on Cloud or self-hosting).

## Startup Prompt

Assuming you're using Anthropic models (which the default model pack does), you'll be asked if you want to connect your Claude subscription the first time you run Plandex. Follow the instructions to connect.

## CLI Commands

### `connect-claude`

You can connect your subscription with the `connect-claude` command.

```bash
plandex connect-claude # CLI
\connect-claude  # REPL
```

### `disconnect-claude`

You can disconnect your subscription and clear credentials from your device with the `disconnect-claude` command.

```bash
plandex disconnect-claude # CLI
\disconnect-claude  # REPL
```

## Quota Exhaustion

If you're using Plandex Cloud with Integrated Models Mode, Anthropic model calls will use your Claude subscription until it runs out of quota, then switch to using Plandex credits until the quota resets.

If you're self-hosting or using Plandex Cloud in BYO API Key Mode, Anthropic model calls will use your Claude subscription until it runs out of quota, then:

- If you have an API key or credentials set for an [Anthropic provider](./model-providers.md) (like the Anthropic API, Google Vertex AI, AWS Bedrock, or OpenRouter), Plandex will switch to the backup provider until the quota resets.

- If you have no API key or credentials set for an Anthropic provider, you'll get a rate limit error until your quota resets.