# Analytics

Koi Cloud measures its own funnel with PostHog (EU): visit → sign-up → trial → paid, retention at
14 and 30 days, and churn. PostHog also hosts the support chat and the setup recordings.

Analytics are optional, like every integration beyond Cloro. Without `KOI_POSTHOG_KEY` the server
sends nothing, `/ingest` does not exist, `/capabilities` returns `analytics: null`, and the app
loads no analytics code. Self-hosted instances keep it that way. Only the production app and the
production landing build get the key, so test data never reaches the project.

## Where events come from

| Source | Code | Events |
| --- | --- | --- |
| Server | `domain.Analytics`, sent by `pkg/providers/posthog` | Sign-up, setup steps, billing, MCP tool calls, finished runs |
| App | `ui/src/analytics`, the only code that imports `posthog-js` | Page views, support chat, setup recordings |
| Landing | the same layer from `ui/site/entry-client.tsx` | Page views with UTM and referrer |

Server events never depend on the browser, so blockers and declined consent do not hide sign-ups
or payments. Billing events are collected inside the transaction and sent only after it commits:
a rollback sends nothing, and a webhook Stripe retries is counted once. They go to the
organization's owner.

The `distinct_id` is the user's UUID everywhere. Each server event carries `organization_id`.
No event carries prompt text, answers or competitor names.

## Event catalog

| Event | Sent when | Properties |
| --- | --- | --- |
| `$pageview` | A page of the landing or the app opens | PostHog's own, with UTM and referrer |
| `user_signed_up` | A sign-up commits | |
| `onboarding_step` | Setup shows a step | `step` |
| `trial_started` | The subscription becomes `trialing` | `quantity` |
| `subscription_activated` | It becomes `active`, from a trial or directly | `quantity`, `amount_cents`, `from_trial` |
| `allowance_changed` | An active plan changes its prompts | `from_quantity`, `to_quantity` |
| `payment_failed` | It becomes `past_due` | |
| `subscription_cancel_scheduled` | Cancellation at period end is set | `tenure_days` |
| `subscription_reactivated` | That cancellation is undone | |
| `subscription_ended` | It becomes `canceled` or `incomplete_expired` | `reason` (`trial_expired`, `payment_failed`, `voluntary`), `tenure_days` |
| `mcp_tool_called` | An agent's tool call succeeds | `tool`, `client` |
| `visibility_run_completed` | A run finishes | `answers`, `first_look` |

Founders who work through their agent may never open the app, so `mcp_tool_called` counts as
activity alongside page views. Daily runs are scheduled by Koi, not by the user, so they do not.

## Consent

Analytics use a cookie only with consent. One small banner, shared by the landing and the app,
stores the choice in `koi_consent` on `.getkoi.co` for 180 days.

- Before a choice, PostHog captures nothing and stores nothing (`opt_out_capturing_by_default`).
- **Allow:** PostHog's cookie on `.getkoi.co` links the landing visit to the sign-up.
- **Decline:** `cookieless_mode: "on_reject"` counts anonymous, aggregate page views only.
- Recordings run on the setup route only, with consent, inputs masked and `data-ph-mask` text
  hidden.

Server events store nothing on the device; the privacy policy lists them under legitimate
interest.

## The team's own accounts

`KOI_ANALYTICS_EXCLUDED_EMAILS` lists accounts that send nothing. The browser opts them out when
they sign in, which also covers their later visits to the landing. The server skips their events.
PostHog's internal-user filter hides `localhost` and the team's email as a last line.

## Why `/ingest`

The app's content security policy allows only `'self'`, and blockers drop requests to PostHog's
hosts. The Go server and the landing's nginx forward `/ingest` to PostHog EU from Koi's own
origin, without the session cookie or credentials.

## PostHog project

Project "Koi" in EU. Its settings:

- cookieless server hash mode on, so declined visits are still counted
- authorized URLs `https://getkoi.co` and `https://app.getkoi.co`
- base currency EUR
- recordings mask every input
- internal-user filter on by default
- support (conversations) enabled

The Growth dashboard holds:

- **Visit → sign-up:** landing `$pageview` → `user_signed_up` within 14 days, by `utm_source` and
  by referring domain, which shows traffic from ChatGPT, Perplexity and Gemini.
- **Sign-up → trial:** `user_signed_up` → `onboarding_step` (`website`, `first_answer`) →
  `trial_started`.
- **Trial → paid:** `trial_started` → `subscription_activated` within 8 days; the trial lasts 7.
- **Retention:** `user_signed_up`, then the action "Koi active" (an app `$pageview` or
  `mcp_tool_called`), at 14 and 30 days.
- **Churn:** `subscription_ended` by `reason`.

Revenue and MRR come from the Stripe source in PostHog's data warehouse, with a read-only
restricted key.

Search before the click lives in Google Search Console (domain property `getkoi.co`, sitemap
`https://getkoi.co/sitemap.xml`). PostHog only sees `google.com` as the referrer.
