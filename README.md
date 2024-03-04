## 🌟 Build large features and entire projects with AI.

💻 Plandex is an open source, terminal-based AI programming engine with long-running agents, automatic file updates, versioning, branches, and diff review.

🔄 Enables a tight feedback loop between programmer and AI.

🔮 Helps you churn through your backlog, learn new technologies, get unstuck, and spend less time on tedious tasks.

🧠 Relies on the OpenAI API and requires an `OPENAI_API_KEY` environment variable. Support for Open Source models, Google Gemini, and Anthropic Claude is coming soon.

✅ Supports Mac and Linux, as well as Windows via [Git bash](https://gitforwindows.org) or [WSL](https://learn.microsoft.com/en-us/windows/wsl/about). Runs from a single binary with no dependencies. Works with every programming language and technology.

## Install 📥

curl -s https://plandex.ai/install.sh | bash

## Why Plandex? 🤔

- 🏗️ Go beyond autocomplete to build complex functionality with AI
- 🚫 Stop the mouse-centered, copy-pasting-back-and-forth madness of coding with ChatGPT
- 📑 Manage context efficiently in the terminal
- ⚡️ Ensure the AI model is always working with the latest version of your files
- 🚧 Experiment, revise, and review in a protected sandbox before applying changes
- ⏪ Rewind and retry as needed with version control
- 🌱 Explore multiple approaches with branches
- 🏎️ Work on multiple tasks in parallel
- 🌡️ Try different models and model settings

## Get started 🚀

```
plandex new
```

To type a bit less, you can use the `pdx` alias instead of `plandex` if you like:

```
pdx new
```

After any plandex command is run, additional commands that may make sense to run next will be suggested.

To see all available commands, run:

```
plandex help
```

For help on any command, run:

```
plandex [command] --help
```

## Plandex Cloud ☁️

Plandex Cloud is the easiest and most reliable way to use Plandex. You'll be prompted to start an anonymous trial (no email required) when you create your first plan with `plandex new`. Anonymous trial accounts are limited to 10 plans and 10 replies per plan. You can upgrade to a full account at any time with a name and email.

Plandex Cloud accounts are free for now. In the future, it will have a monthly fee comparable to other popular AI co-pilot tools.

## Self-hosting 🏠

### Anywhere

The Plandex server runs from a Dockerfile at `app/Dockerfile.server`. It requires a PostgreSQL database and these environment variables:

```bash
export DATABASE_URL=postgres://user:password@host:5432/plandex # replace with your own database URL
export GOENV=production
```

Authentication emails are sent through AWS SES, so you'll need an AWS account with SES enabled. You'll be able to sub in SMTP credentials in the future (PRs welcome).

### AWS

Run `./infra/deploy.sh` to deploy a production-ready Cloudformation stack to AWS.

It requires an AWS account in ~/.aws/credentials, and these environment variables:

```bash
AWS_PROFILE=your-aws-profile
AWS_REGION=your-aws-region
NOTIFY_EMAIL=your-email # AWS cloudwatch alerts and notifications
NOTIFY_SMS=country-code-plus-full-number # e.g. +14155552671 | for urgent AWS alerts
CERTIFICATE_ARN=your-aws-cert-manager-arn # for HTTPS -- must be a valid certificate in AWS Certificate Manager in the same region
```

### Locally

To run the Plandex server locally, run it in development mode with `./dev.sh`. You'll need a PostgreSQL database running locally as well as these environment variables:

```bash
export DATABASE_URL=postgres://user:password@localhost:5432/plandex # replace with your own local database URL
export GOENV=development
```

Authentication codes will be copied to your clipboard with a system notification instead of being sent by email.

To use the `plandex` CLI tool with a local server, first set the `PLANDEX_ENV` environment variable to `development` like this:

```bash
export PLANDEX_ENV=development
plandex new
```

## Security 🔐

Plandex follows best practices for network and data security. As I'm also the founder of a devops-security company ([EnvKey](https://envkey.com)), this is an area I have experience in and take extremely seriously. Data is encrypted in transit and at rest. The database runs within a private, hardened network.

### Ignore sensitive files

Plandex respects .gitignore and won't load any files that you're ignoring. You can also add a .plandexignore file with ignore patterns to any directory.

### API key security

Plandex is a bring-your-own-API-key tool. On the server, API keys are only stored ephemerally in RAM while they are in active use. They are never written to disk, logged, or stored in a database.

It's up to you to manage your API keys securely. Try to avoid storing them in multiple places, exposing them to third party services, or sending them around in plain text. If you'd like some help, please do check out the aforementioned [EnvKey](https://envkey.com). It's open source, end-to-end encrypted, easy to use, and free for up to 3 users. To set your `OPENAI_API_KEY` with EnvKey, you'd add it to an app in the EnvKey UI or CLI, then run `eval $(envkey-source)` in your terminal.

### Third party services

Plandex Cloud relies on AWS for all database and hosting services, Github for code storage, and the OpenAI API for AI models. No other third party services are used.

## Privacy and data retention 🛡️

Data you send to Plandex Cloud is retained in order to debug and improve Plandex. In the future, this data may also be used to train and fine-tune models to improve performance.

That said, if you delete a plan or delete your account, all associated data will be removed from the database. It will still be included in database backups for up to 7 days, then it will no longer exist anywhere on our servers.

If you self-host Plandex, no data will be sent to Plandex Cloud for any reason.

Data sent to the OpenAI API is subject to OpenAI's privacy policy.

## Roadmap 🗺️

- Support for Open Source models, Google Gemini, and Anthropic Claude in addition to OpenAI ✨
- Plan sharing and team collaboration 🤝
- GPT4-Vision support; add images and screenshots to context 🖼️
- VSCode extension 💻
- Github integration 🔌
- Web dashboard and GUI 🌐

This list will grow and be prioritized based on your feedback.

## Discord and discussion 💬

Speaking of feedback, feel free to give yours, ask questions, report a bug, or just hang out:

- Discord
- Discussions
- Issues

## Contributors 👥

Contributors are welcomed, celebrated, and high fived a lot 🙌

Please star ⭐, fork ⑂, explore 🔍, and contribute 🤝

## Comparable tools 🛠️

- [Aider](https://github.com/paul-gauthier/aider)
- [Mentat](https://github.com/AbanteAI/mentat)
- [Pythagora Gpt-pilot](https://github.com/Pythagora-io/gpt-pilot)
- [Sourcegraph Cody](https://github.com/sourcegraph/cody)
- [Continue.dev](https://github.com/continuedev/continue)
- [Sweep.dev](https://github.com/sweepai/sweep)
- [Cursor](https://github.com/getcursor/cursor)
- [Github Copilot](https://github.com/features/copilot)
- [Replit Ghostwriter](https://replit.com/ai)

## About the developer 👋

Hi, I'm Dane. I've been building and launching software products for 17 years. I went through YCombinator in winter 2018 with my devops security company, EnvKey, which I continue to run today. I'm fascinated by LLMs and their potential to transform the practice of software development.

I live with my wife and 4 year old daughter on the SF peninsula. I grew up in the Finger Lakes region of upstate New York. I like reading fiction, listening to podcasts, fitness, and surfing. I started Brazilian Jiu-Jitsu recently and am pretty absorbed with that these days as well.

## Possible co-founder? 😎

I'm looking for a technical co-founder or two with Golang/Devops/TypeScript experience to help me get Plandex off the ground as an open source project, a product, and a fun, WFH-friendly company. If you're interested, please reach out or jump in and start contributing.

## Possible employee? 👩‍💻

While I'm not currently hiring for Plandex, I hope to in the future. If you're an experienced Golang or TypeScript engineer and are interested in working remotely on Plandex for a salary at some point with a group of smart, nice, and fun people, please reach out or jump in and start contributing.

## Possible investor? 💰

I'd love for Plandex's users and contributors to own a significant share of the cap table. Please reach out if you're an angel investor and are interested in investing.
