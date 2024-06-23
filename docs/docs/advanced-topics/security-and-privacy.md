# Security and Privacy

Plandex takes security and privacy seriously. This document outlines the security measures and privacy policies in place for Plandex.

## Security

### Network and Data Security

Plandex Cloud follows best practices for network and data security. Data is encrypted in transit and at rest. The database runs within a private, hardened network.

### API Key Security

Plandex is a bring-your-own-API-key tool. On the Plandex server, whether that's Plandex Cloud or a self-hosted server, API keys are only stored ephemerally in RAM while they are in active use. They are never written to disk, logged, or stored in a database. As soon as a plan stream ends, the API key is removed from memory and no longer exists anywhere on our servers.

### Ignoring Sensitive Files

Plandex respects `.gitignore` and won't load any files that you're ignoring. You can also add a `.plandexignore` file with ignore patterns to any directory.

## Privacy

### Plandex Cloud

Data you send to Plandex Cloud is retained in order to debug and improve Plandex. In the future, this data may also be used to train and fine-tune models to improve performance.

If you delete a plan or delete your Plandex Cloud account, all associated data will be removed. It will still be included in backups for up to 7 days, then it will no longer exist anywhere on Plandex Cloud.

### Third Party Services

Data sent to Plandex Cloud may be shared with the following third parties:

- AWS for hosting and database services. Data is encrypted in transit and at rest.
- In the future, your name and email may be shared with a third party transactional email service and/or email marketing service to send you information about your account or updates on Plandex. You will be able to opt out of these emails at any time.
- When billing and payments are implemented, your name and email may be shared with our payment processor Stripe.
- In the future, your name, email, and metadata on the actions you take with Plandex may be shared with a third party analytics service to track usage and make improvements.

Apart from the above list, no other data will be shared with any other third party. The list will be updated when a specific third party is chosen for any of the above services, or if any new third party services are introduced.

Data sent to a model provider like OpenAI is subject to the model provider's privacy and data retention policies.

### Self-Hosting

If you self-host Plandex, no data will be sent to Plandex Cloud or to any third party, except for the data you send to model providers like OpenAI.
