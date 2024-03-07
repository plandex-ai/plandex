# Plandex self-hosting 🏠

## Anywhere

The Plandex server runs from a Dockerfile at `app/Dockerfile.server`. It requires a PostgreSQL database (ideally v14) and these environment variables:

```bash
export DATABASE_URL=postgres://user:password@host:5432/plandex # replace with your own database URL
export GOENV=production
```

The server listens on port 8088 by default.

Authentication emails are sent through AWS SES, so you'll need an AWS account with SES enabled. You'll be able to sub in SMTP credentials in the future (PRs welcome).

## AWS

Run `./infra/deploy.sh` to deploy a production-ready Cloudformation stack to AWS with built-in autoscaling, clustering, network security, secrets management, backups, and alerts. This is the same infrastructure Plandex Cloud runs on.

Fair warning: this will cost signficantly more than paying for Plandex Cloud (once billing is enabled) or self-hosting with a lighter setup.

It requires an AWS account in ~/.aws/credentials, and these environment variables:

```bash
export AWS_PROFILE=your-aws-profile
export AWS_REGION=your-aws-region
export NOTIFY_EMAIL=your-email # AWS cloudwatch alerts and notifications
export NOTIFY_SMS=country-code-plus-full-number # e.g. +14155552671 | for urgent AWS alerts
export CERTIFICATE_ARN=your-aws-cert-manager-arn # for HTTPS -- must be a valid certificate in AWS Certificate Manager in the same region

./infra/deploy.sh
```

To deploy a new build when only the app code has changed (not the infrastructure), run `./infra/deploy.sh --image-only`

If the infrastructure _has_ changed, run `./infra/deploy.sh` again.

## SES sandbox

Note that in order to send authentication emails with SES, you'll either need to make a request to AWS get out of the SES sandbox, verify an email domain, or verify email addresses individually. [Read more here.](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/request-production-access.html)

## Create a new account

Once the server is running, you can create a new account by running `plandex sign-in` on your local machine.

```bash
plandex sign-in # follow the prompts to create a new account on your self-hosted server
```
