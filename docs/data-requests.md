# Requests about personal data

Koi has no self-service page for these requests. An administrator answers them with the commands
below, which read `KOI_DATABASE_URL` from the environment. Have your lawyer review this procedure
together with the Privacy Policy and Terms.

## Before you act

1. Confirm the request comes from the account's email address. Reply to that address and act only
   on the answer. Do not act on a request sent from anywhere else.
2. Answer within one month, free of charge.

## A copy of the data

```bash
koi admin export-account person@example.com > export.json
```

The file holds the account, organization, subscription state, onboarding steps, connected agents,
brands, prompts, competitors, drafts, runs and answers. It holds no password hash or token. Send
it through a channel you trust and delete your copy afterwards.

## Deletion

```bash
koi admin delete-account person@example.com --yes
```

This removes the account, its organization and all of the above in one transaction, along with
queued emails and jobs, and then ends its sessions.

The command refuses when:

- **The Stripe subscription is not finished.** Cancel it in Stripe first and wait for the webhook.
- **The organization has other members.** Decide with them what should happen first.
- **A draft or collection is running.** Try again when it ends.

The command prints the Stripe customer when there is one. Delete it in the Stripe dashboard, and
keep what tax law requires you to keep, such as invoices.

## What remains

- **Backups** keep the data until they expire. Tell the person how long that takes, and delete the
  account again if a backup is ever restored.
- **Subprocessors** are not reached by these commands. Prompts and engine answers have passed
  through Cloro and OpenRouter, and email addresses through Resend. Follow each one's process if
  the request covers them. List them in the Privacy Policy, along with the fact that the brand
  lookup reads the site the person entered.
