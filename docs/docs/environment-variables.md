---
sidebar_position: 11
sidebar_label: Environment Variables
---

# Environment Variables

This is an overview of all the environment variables that can be used with Plandex.

## CLI

### General

```bash
PLANDEX_ENV=development # Set this to 'development' to default to the local development server instead of Plandex Cloud when working on Plandex itself.
PLANDEX_API_HOST= # Defaults to 'http://localhost:8099' if PLANDEX_ENV is development, otherwise it's 'https://api.plandex.ai'—override this to use a different host.
```

### LLM Providers

```bash
# OpenRouter.ai
OPENROUTER_API_KEY= # Your OpenRouter.ai API key 

# OpenAI
OPENAI_API_KEY= # Your OpenAI key 
OPENAI_ORG_ID= # Your OpenAI organization ID. Defaults to empty.

# Anthropic
ANTHROPIC_API_KEY= # Your Anthropic API key 

# Google AI Studio
GEMINI_API_KEY= # Your Google AI Studio API key 

# Google Vertex AI
GOOGLE_APPLICATION_CREDENTIALS= # Your Google Vertex AI credentials file path
VERTEXAI_PROJECT= # Your Google Vertex AI project ID
VERTEXAI_LOCATION= # Your Google Vertex AI location

# Azure OpenAI
AZURE_OPENAI_API_KEY= # Your Azure OpenAI API key
AZURE_API_BASE= # Your Azure OpenAI API base URL
AZURE_API_VERSION= # Your Azure OpenAI API version
AZURE_DEPLOYMENTS_MAP= # Your Azure OpenAI deployments map—a JSON object mapping model names to deployment names (only needed if deployment names are different from model names)

# DeepSeek
DEEPSEEK_API_KEY= # Your DeepSeek API key

# Perplexity
PERPLEXITY_API_KEY= # Your Perplexity API key

# Amazon Bedrock
PLANDEX_AWS_PROFILE= # Name of AWS profile in ~/.aws/credentials to use for AWS Bedrock. If not set, the credentials file won't be used.
AWS_ACCESS_KEY_ID= # Your AWS access key ID
AWS_SECRET_ACCESS_KEY= # Your AWS secret access key
AWS_REGION= # Your AWS region
AWS_SESSION_TOKEN= # Your AWS session token
AWS_INFERENCE_PROFILE_ARN= # Your AWS inference profile ARN
```

### Upgrades

```bash
PLANDEX_SKIP_UPGRADE= # Set this to '1' to skip the auto-upgrade check when running the CLI.
```

### Development

Check out the [Development Guide](./development.md) for more details.

```bash
PLANDEX_OUT_DIR=/usr/local/bin # Where the development binary should be output when using dev.sh
PLANDEX_DEV_CLI_OUT_DIR=/usr/local/bin # Where the development binary should be output when using dev.sh
PLANDEX_DEV_CLI_NAME=plandex-dev # The name of the development binary when using dev.sh
PLANDEX_DEV_CLI_ALIAS=pdxd # The alias for the development binary when using dev.sh
GOPATH= # This should be already set to your Go folder if you've installed Golang.
```

## Server

Check out the [Self-Hosting Guide](./hosting/self-hosting/local-mode-quickstart.md) for more details.

### General

```bash
GOENV=development # Whether to run in development or production mode. Must be 'development' or 'production'
PLANDEX_BASE_DIR= # The base directory to read and write files. Defaults to '$HOME/plandex-server' in development mode, '/plandex-server' in production.
API_HOST= # The host the API server listens on. Defaults to 'http://localhost:$PORT'. In production mode, should be a host like 'https://api.your-domain.ai'.
PORT=8099 # The port the server listens on. Defaults to 8099.
```

### docker-compose

For self-hosting with docker-compose, default environment variables are set in `app/_env`. This file should be copied to `app/.env` before running the server. You can override any of these defaults in `.env`. 

```bash
PLANDEX_DATA_DIR=/var/lib/plandex/data # When using docker-compose, this is the directory *on your machine* that the Plandex server will use to store data—it will be mounted to the Docker container as a volume.

# Database Credentials
POSTGRES_DATABASE=plandex # Your postgres database.
POSTGRES_USER=plandex # Your postgres user.
POSTGRES_PASSWORD=plandex # Your postgres password.
```

### Other methods

If you're *not* using docker-compose, you'll need a `DATABASE_URL` environment variable that points to a PostgreSQL database. For example, if you're running PostgreSQL locally, you might set it to something like this:

```bash
DATABASE_URL=postgres://plandex:<password>@<host>:<port>/plandex?sslmode=disable
```

If you're running in production mode, you'll also need to set `API_HOST` to the host the API server is running on.

```bash
API_HOST= https://api.your-domain.ai # The host of the API server in production mode. Defaults to 'http://localhost:$PORT' in development mode.
```


### SMTP

If you're running in production mode (with `GOENV=production`, typically on a remote server), you'll need SMTP credentials:

```bash
SMTP_HOST= # Your SMTP host.
SMTP_PORT= # Set this to 1025 e.g. if you are using mailhog.
SMTP_USER= # SMTP username.
SMTP_PASSWORD= # SMTP password.
```
