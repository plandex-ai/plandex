# Self-Hosting

Plandex can be self-hosted for greater control and customization. Follow these steps to set up a self-hosted Plandex server.

## Quick Start Script

To quickly set up a self-hosted Plandex server, run the following commands:

```bash
git clone https://github.com/plandex-ai/plandex.git
cd plandex/app
./start_local.sh
```

## Requirements

The Plandex server requires a PostgreSQL database (ideally v14), a persistent file system, git, and the following environment variables:

```bash
export DATABASE_URL=postgres://user:password@host:5432/plandex
export GOENV=production
export SMTP_HOST=smtp.example.com
export SMTP_PORT=587
export SMTP_USER=user
export SMTP_PASSWORD=password
export SMTP_FROM=user@example.com  # optional, if not set then SMTP_USER is used
```

## Using Docker Build

To run the Plandex server from a Dockerfile, follow these steps:

```bash
git clone https://github.com/plandex-ai/plandex.git
VERSION=$(cat app/server/version.txt)  # or use the version you want
git checkout server/v$VERSION
cd plandex/app
mkdir ~/plandex-server  # or another directory where you want to store files
docker build -t plandex-server -f Dockerfile.server .
docker run -p 8080:8080 \
  -v ~/plandex-server:/plandex-server \
  -e DATABASE_URL \
  -e GOENV \
  -e SMTP_HOST \
  -e SMTP_PORT \
  -e SMTP_USER \
  -e SMTP_PASSWORD \
  plandex-server
```

## Using Docker Compose

If you don't have a PostgreSQL server, you can use the `docker-compose.yml` file:

```bash
cd plandex/app
cp _env .env
# edit .env to set the required environment variables for postgres
docker compose build
docker compose up
```

## Running from Source

To run the Plandex server from source, follow these steps:

```bash
git clone https://github.com/plandex-ai/plandex.git
cd plandex/
VERSION=$(cat app/server/version.txt)  # or use the version you want
git checkout server/v$VERSION
cd app/server
export PLANDEX_BASE_DIR=~/plandex-server  # or another directory where you want to store files
go run main.go
```

## Development Mode

In development mode, authentication tokens will be copied to the clipboard instead of sent via email, and a system notification will pop up to let you know that the token is ready to paste. The pin will also be output to the console.

To set up development mode, set the following environment variable:

```bash
export GOENV=development
```

## Health Check

You can check if the server is running by sending a GET request to `/health`. If all is well, it will return a 200 status code.

## Creating a New Account

Once the server is running, you can create a new account by running the `plandex sign-in` command on your local machine:

```bash
plandex sign-in
```

## Local CLI Files

If you reset the database or use a new one, you'll need to remove the local files that the CLI creates in directories where you used Plandex to start fresh. Remove the following in any directory you used the CLI in:

- `.plandex-dev` directory if you ran the CLI with `PLANDEX_ENV=development`
- `.plandex` directory otherwise

Then run `plandex sign-in` again to create a new account.

If you're still having trouble with accounts, you can also remove the following from your $HOME directory to fully reset them:

- `.plandex-home-dev` directory if you ran the CLI with `PLANDEX_ENV=development`
- `.plandex-home` directory otherwise
