# Plandex self-hosting 🏠

## Anywhere

The Plandex server runs from a Dockerfile at `app/Dockerfile.server`. It requires a PostgreSQL database (ideally v14) and these environment variables:

```bash
export DATABASE_URL=postgres://user:password@host:5432/plandex # replace with your own database URL
export GOENV=production
```

The server listens on port 8088 by default.

The server also requires access to a persistent file system. It should be mounted to the container. In production, the `/plandex-server` directory is used by default as the base directory to read and write files. You can use the `PLANDEX_BASE_DIR` environment variable to change this.

Authentication emails are sent through AWS SES in production, so you'll need an AWS account with SES enabled. You'll be able to sub in SMTP credentials in the future (PRs welcome).

If you set `export GOENV=development` instead of `production`:

- Authentication tokens will be copied to the clipboard instead of sent via email, and a system notification will pop up to let you know that the token is ready to paste.

- The default base directory will be `$HOME/plandex-server` instead of `/plandex-server`. It can still be overridden with `PLANDEX_BASE_DIR`.

## Create a new account

Once the server is running, you can create a new account by running `plandex sign-in` on your local machine.

```bash
plandex sign-in # follow the prompts to create a new account on your self-hosted server
```
