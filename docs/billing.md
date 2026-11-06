# Koi Cloud billing (local implementation)

Koi Cloud bills one organization for a monthly allowance of active tracked prompts. All
features, brands, members and supported engines are included. Self-hosted installations
use `KOI_DEPLOYMENT_MODE=selfhosted` (the default) and never call Stripe. They still
need their own Cloro key for live collection.

## Pricing

`pkg/pricing/catalog.json` contains the 10 selectable tax-inclusive monthly allowances:
15 (€29), 25 (€45), 50 (€75), 75 (€99), 100 (€129), 150 (€179), 200 (€229),
250 (€269), 350 (€349), and 500 (€449).
The server quotes in integer cents and rejects other quantities, including 5 and 6.
There is no catalog version. One Stripe volume-tiered monthly Price holds all
allowances. Each tier has only a `flat_amount`; its `unit_amount` is omitted.
The subscription item quantity is the selected prompt allowance.
Stripe Checkout shows the selected tax-inclusive total without a zero-priced
unit row. Stripe's invoice preview still renders a separate zero-priced unit
line and the tier's flat-fee line for this volume-tiered Price. Omitting
`unit_amount` does not remove that invoice presentation.

## Sandbox setup

Do these steps in a Stripe **sandbox** before enabling Cloud mode. The local code does
not create or modify objects in the account automatically at startup.

Enable [Managed Payments in the Stripe Dashboard](https://dashboard.stripe.com/settings/managed-payments)
for the sandbox before testing Checkout, and separately for the live account before
accepting live payments. The SaaS product tax code alone does not enable Stripe as
merchant of record. Koi sends `managed_payments[enabled]=true` on every new Checkout
Session, including trials, using the pinned Stripe Go SDK. It omits the unsupported
tax and customer-update parameters. This applies to new subscriptions purchased
through these sessions; existing subscriptions are not converted. See
[Stripe's Checkout integration guide](https://docs.stripe.com/payments/managed-payments/update-checkout).

1. Log in with the installed CLI: `stripe login`.
2. Use the sandbox Product `prod_VTTHrgW2E8Lgny`, named `Koi Cloud`.
   Managed Payments requires an eligible product tax code; this Product uses
   `txcd_10103001` (SaaS for business use). In a new account:

   ```sh
   stripe products create --name "Koi Cloud" --tax-code txcd_10103001
   ```

   Choose the live Product's tax code according to the actual offering and tax setup.

3. Create one EUR monthly **licensed, volume-tiered** Price with a `flat_amount`
   per allowance, `tax_behavior=inclusive`, and no `unit_amount` field in any
   tier. Koi rejects prices with explicit zero unit prices, exclusive taxes,
   or tiers that differ from the catalog. The command below is a safe preview:

   ```sh
   stripe prices create --dry-run --product prod_VTTHrgW2E8Lgny --currency eur \
     --billing-scheme tiered --tiers-mode volume \
     --recurring.interval month --tax-behavior inclusive \
     -d 'tiers[0][up_to]=15' -d 'tiers[0][flat_amount]=2900' \
     -d 'tiers[1][up_to]=25' -d 'tiers[1][flat_amount]=4500' \
     -d 'tiers[2][up_to]=50' -d 'tiers[2][flat_amount]=7500' \
     -d 'tiers[3][up_to]=75' -d 'tiers[3][flat_amount]=9900' \
     -d 'tiers[4][up_to]=100' -d 'tiers[4][flat_amount]=12900' \
     -d 'tiers[5][up_to]=150' -d 'tiers[5][flat_amount]=17900' \
     -d 'tiers[6][up_to]=200' -d 'tiers[6][flat_amount]=22900' \
     -d 'tiers[7][up_to]=250' -d 'tiers[7][flat_amount]=26900' \
     -d 'tiers[8][up_to]=350' -d 'tiers[8][flat_amount]=34900' \
     -d 'tiers[9][up_to]=inf' -d 'tiers[9][flat_amount]=44900'
   ```

   The final tier must have `up_to=inf`; Koi rejects quantities above 500.
   The verified sandbox Price in account
   `acct_1URAU0Gf1728FFn1` is:

   ```dotenv
   KOI_STRIPE_PRICE_ID=price_1UYLJ0Gf1728FFn1ZjJ9NKiS
   ```

   Create a separate live Price before enabling live billing. Sandbox IDs cannot
   be used with live keys.

After creating and validating the Price:

4. Configure a Customer Portal in the Stripe Dashboard for payment methods and invoices.
   Keep plan/quantity changes in Koi. If cancellation is enabled, select cancellation
   at the end of the paid period.
5. Run `stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe` and set
   `KOI_STRIPE_WEBHOOK_SECRET` to the `whsec_...` printed by that command. Use an actual
   Koi Checkout session when testing billing identity; synthetic `stripe trigger`
   sessions have no matching organization reservation in Koi's database.
6. Set `KOI_DEPLOYMENT_MODE=cloud` and `KOI_PUBLIC_URL` to the browser origin. For
   local Vite development use `http://localhost:5173`; for a hosted environment use
   HTTPS. In development Vite proxies `/api` to the Go server.

Cloud mode requires `KOI_CLORO_API_KEY`, `KOI_STRIPE_SECRET_KEY`,
`KOI_STRIPE_WEBHOOK_SECRET`, `KOI_STRIPE_PRICE_ID`, and `KOI_PUBLIC_URL` in
addition to the common `KOI_PORT`, `KOI_DATABASE_URL`, and `KOI_SESSION_SECRET`.
The default self-hosted mode does not require Cloro or Stripe credentials.
Startup checks that the Cloro key is present, but does not contact Cloro to verify it.

The account must be activated separately for live payments, with Stripe's business
verification, payout and tax settings completed in its Dashboard. Use live API keys,
a live Price and a live webhook signing secret only after sandbox flows pass. Sandbox
IDs and secrets are not interchangeable with live IDs and secrets.

## Data flow

Koi creates one Stripe Customer per organization and stores its ID before Checkout.
It creates a **hosted** Checkout Session with `customer`,
`client_reference_id=organization_id`, one Price and the selected prompt quantity.
The browser receives only the Stripe-hosted redirect URL. It never constructs a
Checkout URL or supplies a price or entitlement. A durable database reservation serializes Checkout requests for the organization;
retries reuse the same session and finalization accepts the same session repeatedly.
Choosing a different allowance expires the open session before replacing it. If payment
wins the race with expiration, Koi reconciles that subscription and prevents a second purchase.
The success redirect includes the Stripe Checkout Session ID. Koi verifies that saved session
directly with Stripe and reconciles it when complete. The maintenance worker also checks
unfinished sessions, so a closed browser or a missing local webhook listener does not strand
a completed checkout. These paths use the same organization lock and validation as webhook
processing; the redirect alone never grants access.

`checkout.session.completed` records the subscription ID only when the signed event's
session ID, customer ID and organization ID match the reserved session and organization.
Subscription events are reconciled only against that completed Checkout and the
organization's stored Customer. The organization is locked before retrieving the current subscription from Stripe,
including for deletion events. Each event ID, Checkout association and subscription update
is committed in the same transaction, so replay is harmless and concurrent reads cannot
overwrite a newer local state. Premature subscription events are ignored; the Checkout
event fetches the current subscription to complete synchronization. Delayed subscription
events from a previous subscription cannot replace its successor.
Email is never used as an identity key.

### First Look

An organization with no subscription and an unused trial gets one free check before any payment.
It can draft up to 5 prompts and run one collection, and it keeps reading those results. Later
collections are blocked until Checkout, and `tracking_get` reports `first_look_used`; the daily
dispatcher then resumes on the next slot without any other step. Canceled subscriptions and used
trials never get it. `BillingStatus.firstLook` exposes the state to the UI and to MCP.

Sign-up has no email verification or rate limit, so two limits bound the spend. A website or
repository gets one First Look across organizations, and `KOI_FIRST_LOOK_DAILY_LIMIT` (default 100)
caps the checks started in 24 hours. Either limit reports `first_look_unavailable`, and the
organization can still start its trial.

Setup asks the draft to start collection (`GenerationRequest.startTracking`), so the worker starts
the check when the draft succeeds and closing the page cannot stop it. An agent does not set it,
because it may still be correcting prompts, and calls `tracking_start` itself. When the first check
finishes with at least one real answer, the owner gets a "ready" email if Resend is configured.

The database holds the application entitlement. `active` subscriptions grant their
selected allowance until the current period ends. `trialing` grants 10 prompts
until trial end, with a payment method collected in Checkout. A brand's one AI draft stores
at most 5 prompts on any plan, leaving room for the customer's own. Other statuses grant zero;
data is preserved after payment failure or cancellation. An immediate upgrade uses Stripe proration and a pending update, so the larger
entitlement arrives only after Stripe applies the paid change. A pending upgrade includes
the hosted invoice URL and expiration time so the customer can recover a declined payment
or complete authentication. Subscription updates and pending-update applied/expired events
reconcile these fields. A downgrade is scheduled for the next cycle. It can be replaced or
canceled by choosing the current allowance; upgrading releases an existing schedule first.
During a trial, changing the future allowance updates quantity without proration and
preserves the trial end date and 10-prompt entitlement. If the
new limit is below the active prompt count, tracking stops until enough prompts are
paused. All prompts remain stored. Cancellation via Portal retains access until the
period end when Stripe reports `cancel_at_period_end` and `active`.

### Starting the plan during a trial

`POST /api/v1/billing/start-plan` requires `Idempotency-Key`, takes no body and charges the
plan already selected. It updates the subscription with `trial_end=now`,
`payment_behavior=pending_if_incomplete` and `proration_behavior=none`. The billing date moves
to today and unused trial days are not credited.

- Payment succeeds: returns 200 with `effective=started`, and the allowance applies at once.
- Payment fails or needs authentication: returns 202 with `effective=after_payment`,
  `payment_url` and `expires_at`. The subscription stays `trialing` with the trial allowance
  and is not `past_due`. Stripe holds this pending update for 23 hours, then voids the invoice.
  Koi exposes it through the existing pending upgrade fields, with the current quantity as
  target, and allowance changes are rejected while it is open.

A new Stripe attempt voids the open invoice and its link, so a repeated request returns the
open payment without calling Stripe.

Managed Payments subscriptions cannot change `default_payment_method` through the API, and
`invoices/pay` accepts no other payment method. Recovery therefore uses the hosted invoice
page. Paying there with a new card does not replace the subscription's default card: in the
sandbox the next renewal retried the declined card and went `past_due`. Customers change the
card in the Portal, and the pending message tells them to.

## Application boundaries and API

`BillingCommander` owns checkout, allowance changes, Portal access and webhook reconciliation.
`BillingQuerier` exposes the catalog, quote, subscription status and product access. Both use
`Store.BillingRepo()`; `stripe.Client` implements the domain's `BillingGateway` and owns SDK
mapping. HTTP handlers decode requests, verify signatures and return typed DTOs.

- `GET /api/v1/billing/catalog` returns `{currency, interval, options}`. Each option contains
  `prompts`, `amount_cents`, `currency` and `interval`. It validates the configured Stripe Price
  before exposing the embedded catalog. The UI derives its choices and displayed prices here.
- `POST /api/v1/billing/change-allowance` requires `Idempotency-Key` and `{prompts}`. Koi stores
  the operation before calling Stripe, serializes changes per organization and uses separate
  Stripe idempotency keys for each remote step. Repeating a key returns its original result;
  changing its quantity returns `idempotency_conflict`. Unfinished operations are recovered in
  creation order, each in its own transaction, before a later change is applied.
- The change response has `quantity`, `effective` (`current`, `trial_end`, `next_cycle` or
  `after_payment`), and optional `payment_url` and `expires_at`. A no-op or canceled downgrade
  returns 200; submitted changes return 202.
- Subscription status includes `pending_quantity`, `pending_upgrade_quantity`,
  `pending_upgrade_expires_at` and `payment_url`. The UI polls for checkout/upgrade confirmation
  for up to 60 seconds, then offers a manual refresh and payment recovery.

Billing uses organization-scoped query keys. Self-hosted mode constructs no Stripe client or
billing service. The schema is initialized on a fresh database, including checkout reservations,
change requests and processed events; there are no historical schema migrations.

## Internal collection cost

Each visibility result records the engine, collector, submission time and, when
Cloro supplies `X-Credits-Charged` on the completed task response, the credits charged.
The first visibility run of an organization uses synchronous Cloro requests for faster
initial results; subsequent runs use async tasks. Cloro currently charges two extra
credits per synchronous request. A timed-out synchronous request may have completed
at Cloro, so its one allowed retry can consume credits twice.
This supports internal execution and credit reports without affecting the customer
price. Cloro credits are the provider cost unit; converting them to money requires
the operator's current Cloro plan and invoice because its plans have monthly credit
allowances. Do not infer a fixed euro cost per task from this field.

Example internal query for a UTC month:

```sql
SELECT b.organization_id, vr.engine,
       count(*) FILTER (WHERE vr.submitted_at IS NOT NULL) AS submissions,
       count(*) FILTER (WHERE vr.outcome IS NOT NULL) AS completed_cells,
       sum(vr.provider_credits) AS reported_credits
FROM visibility_results vr
JOIN visibility_runs run ON run.id = vr.run_id
JOIN brands b ON b.id = run.brand_id
WHERE vr.created_at >= '2026-10-01' AND vr.created_at < '2026-11-01'
GROUP BY b.organization_id, vr.engine;
```

## Sandbox acceptance checks

- Quote and Checkout for all 10 allowances match the same tax-inclusive Stripe amounts. Other
  quantities, including 5, 6 and 501, are rejected.
- Two browser tabs requesting Checkout for one organization reuse one session.
- The webhook ignores a replay and rejects a Customer or organization mismatch.
- Upgrade gains entitlement only after Stripe confirms payment; downgrade changes it
  at renewal. Cancellation keeps access until the paid period ends.
- Stopped prompts do not consume the allowance and already queued cells do not submit
  to Cloro after deactivation.
- A self-hosted install starts and tracks with its own Cloro key and no Stripe keys.

Stripe's invoice preview splits a zero-priced unit line and the tier's flat-fee line.
Repeatable sandbox validation is available with:

```sh
KOI_STRIPE_SANDBOX_TEST=1 go test ./pkg/providers/stripe -run TestStripeSandbox -v
```

The opt-in suite reads the sandbox key and Price from `.env`, refuses a live key, creates
isolated test clocks/customers and deletes those clocks after each scenario. It checks trial
changes, downgrade renewal and release, successful/declined/authentication-required upgrades,
and cancellation at period end. This exercises the same cancellation API behavior expected
from the Portal; Dashboard configuration and the hosted Portal UI must still be checked when
configuring a new Stripe account.

Local checks:

```sh
go test ./...
pnpm --dir ui test
pnpm --dir ui check-types
pnpm --dir ui test:e2e
```

The browser suite mocks API responses and covers existing and pending allowances, payment
recovery, bounded polling, error recovery with the same idempotency key, and a 320px viewport.
