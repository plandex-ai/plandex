---
sidebar_position: 9
sidebar_label: Security
---

# Security
## Ignoring Sensitive Files

Plandex respects `.gitignore` and won't load any files that you're ignoring unless you use the `--force/-f` flag with `plandex load`. You can also add a `.plandexignore` file with ignore patterns to any directory.

## API Key Security

When [self-hosting](./hosting/self-hosting.md) or using [Plandex Cloud](./hosting/cloud.md) in BYO API Key Mode, API keys are only stored ephemerally in RAM while they are in active use. They are never written to disk, logged, or stored in a database. As soon as a plan stream ends, the API key is removed from memory and no longer exists anywhere on the Plandex server.

It's also up to you to manage your API keys securely. Try to avoid storing them in multiple places, exposing them to third party services, or sending them around in plain text.

You may also want to consider using Plandex Cloud in [Integrated Models Mode](./hosting/cloud.md#integrated-models-mode), which lets you skip dealing with API keys at all.