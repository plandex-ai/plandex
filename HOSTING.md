## Plandex self-hosting 🏠

#### Anywhere

The Plandex server runs from a Dockerfile at `app/Dockerfile.server`. It requires a PostgreSQL database (ideally v14) and these environment variables:

```bash
export DATABASE_URL=postgres://user:password@host:5432/plandex # replace with your own database URL
export GOENV=production
```

The server listens on port 8088 by default.

Authentication emails are sent through AWS SES, so you'll need an AWS account with SES enabled. You'll be able to sub in SMTP credentials in the future (PRs welcome).

#### AWS

Run `./infra/deploy.sh` to deploy a production-ready Cloudformation stack to AWS.

It requires an AWS account in ~/.aws/credentials, and these environment variables:

```bash
AWS_PROFILE=your-aws-profile
AWS_REGION=your-aws-region
NOTIFY_EMAIL=your-email # AWS cloudwatch alerts and notifications
NOTIFY_SMS=country-code-plus-full-number # e.g. +14155552671 | for urgent AWS alerts
CERTIFICATE_ARN=your-aws-cert-manager-arn # for HTTPS -- must be a valid certificate in AWS Certificate Manager in the same region
```

#### Locally

To run the Plandex server locally, run it in development mode with `./dev.sh`. You'll need a PostgreSQL database (ideally v14) running locally as well as these environment variables:

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
