# Plandex security 🔐

Plandex Cloud follows best practices for network and data security. As I (Plandex's creator 👋) am also the founder of a devops-security company ([EnvKey](https://envkey.com)), this is an area I have experience in and take extremely seriously. Data is encrypted in transit and at rest. The database runs within a private, hardened network.

## Ignores sensitive files

Plandex respects `.gitignore` and won't load any files that you're ignoring. You can also add a `.plandexignore` file with ignore patterns to any directory.

## API key security

Plandex is a bring-your-own-API-key tool. On the Plandex server, whether that's Plandex Cloud or a self-hosted server, API keys are only stored ephemerally in RAM while they are in active use. They are never written to disk, logged, or stored in a database. As soon as a plan stream ends, the API key is removed from memory and no longer exists anywhere on our servers.

It's also up to you to manage your API keys securely. Try to avoid storing them in multiple places, exposing them to third party services, or sending them around in plain text. If you'd like some help, please do check out the aforementioned [EnvKey](https://envkey.com) for secrets management. It's open source, end-to-end encrypted, easy to use, and free for up to 3 users. To set your `OPENAI_API_KEY` with EnvKey, you'd add it to an app in the EnvKey UI or CLI, then run `eval $(envkey-source)` in your terminal.

## Third party services

- Plandex Cloud relies on AWS for all database and hosting services.
- [EnvKey](https://envkey.com) for configuration and secrets management (only used in development so far).
- Github for code storage.
- OpenAI (called on your behalf) for AI models (other model providers will be added in the future).
- See the [privacy overview](PRIVACY.md) for more details on third parties and data retention.
