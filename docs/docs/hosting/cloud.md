---
sidebar_position: 1
sidebar_label: Cloud
---

# Plandex Cloud

## Overview

Plandex Cloud is the easiest and most reliable way to use Plandex. You'll be prompted to start a trial when you create your first plan with `plandex new`. 

## Billing Modes

Plandex Cloud has two billing modes:

### Integrated Models 

- Use Plandex credits to pay for AI models. 
- No separate accounts or API keys are required. 
- Credits are deducted at the model's price from OpenAI or OpenRouter.ai plus a small markup to cover credit card processing costs.
- Start with a $5 trial (includes $5 in credits).
- After the trial, you can upgrade to a paid plan for $25 per month—includes $10 in credits every month that never expire.

### BYO API Key

- Use your own OpenAI, OpenRouter.ai, or other OpenAI-compatible provider accounts.
- Supply your own API keys.
- Start with a free trial up to 10 plans and 20 model responses per plan.
- After the trial, you can upgrade to a paid plan for $20 per month.

## Billing Settings

Run `plandex billing` in the terminal to bring up the billing settings page in your default browser, or go to [https://app.plandex.ai/settings/billing](https://app.plandex.ai/settings/billing) (sign in if necessary).

Here you can switch billing modes, view your current plan, manage your billing details, and pause or cancel your subscription.

## Integrated Models Mode

If you're using **Integrated Models Mode**, you can use the billing settings page to view your credits balance, purchase credits, and configure auto-recharge settings to automatically add credits to your account when your balance gets too low. You can also set a monthly budget and an email notification threshold.

You can also see your credits balance in the terminal with `plandex credits`.

You can see a full history of your usage that includes every model call and response with `plandex usage`.

## Privacy / Data Retention

Data you send to Plandex Cloud is retained in order to debug and improve Plandex. In the future, this data may also be used to train and fine-tune models to improve performance.

That said, if you delete a plan or delete your Plandex Cloud account, all associated data will be removed. It will still be included in backups for up to 7 days, then it will no longer exist anywhere on Plandex Cloud.

Data sent to Plandex Cloud may be shared with the following third parties:

- [OpenAI](https://openai.com) for OpenAI models when using Integrated Models Mode.
- [OpenRouter.ai](https://openrouter.ai/) for Anthropic, Google, and other non-OpenAI models when using Integrated Models Mode.
- [AWS](https://aws.amazon.com/) for hosting and database services. Data is encrypted in transit and at rest.
- Your name and email is shared with [Loops](https://loops.so/), an email marketing service, in order to send you updates on Plandex. You can opt out of these emails at any time with one click.
- Your name and email are shared with our payment processor [Stripe](https://stripe.com/) if you subscribe to a paid plan or purchase the $5 trial.
- Usage data is sent to [Google Analytics](https://analytics.google.com/) to help us track usage and make improvements.

Apart from the above list, no other data will be shared with any other third party. The list will be updated if any new third party services are introduced.

Data sent to a model provider like OpenAI or OpenRouter.ai is subject to the model provider's privacy and data retention policies.