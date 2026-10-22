# Koi Cloud billing (local implementation)

Koi Cloud bills one organization for a monthly allowance of active tracked prompts. All
features, brands, members and supported engines are included. Self-hosted installations
use `KOI_DEPLOYMENT_MODE=selfhosted` (the default) and never call Stripe. They still
need their own Cloro key for live collection.

## Pricing

`pkg/koipricing/catalog.json` contains the 12 selectable tax-inclusive monthly allowances:
5 (€9), 10 (€15), 15 (€20), 25 (€29), 50 (€49), 75 (€69), 100 (€89),
150 (€119), 200 (€149), 250 (€179), 350 (€235), and 500 (€319).
The server quotes in integer cents and rejects other quantities, including 6.
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
     -d 'tiers[0][up_to]=5' -d 'tiers[0][flat_amount]=900' \
     -d 'tiers[1][up_to]=10' -d 'tiers[1][flat_amount]=1500' \
     -d 'tiers[2][up_to]=15' -d 'tiers[2][flat_amount]=2000' \
     -d 'tiers[3][up_to]=25' -d 'tiers[3][flat_amount]=2900' \
     -d 'tiers[4][up_to]=50' -d 'tiers[4][flat_amount]=4900' \
     -d 'tiers[5][up_to]=75' -d 'tiers[5][flat_amount]=6900' \
     -d 'tiers[6][up_to]=100' -d 'tiers[6][flat_amount]=8900' \
     -d 'tiers[7][up_to]=150' -d 'tiers[7][flat_amount]=11900' \
     -d 'tiers[8][up_to]=200' -d 'tiers[8][flat_amount]=14900' \
     -d 'tiers[9][up_to]=250' -d 'tiers[9][flat_amount]=17900' \
     -d 'tiers[10][up_to]=350' -d 'tiers[10][flat_amount]=23500' \
     -d 'tiers[11][up_to]=inf' -d 'tiers[11][flat_amount]=31900'
   ```

   The final tier must have `up_to=inf`; Koi rejects quantities above 500.
   The verified sandbox Price in account
   `acct_1URAU0Gf1728FFn1` is:

   ```dotenv
   KOI_STRIPE_PRICE_ID=price_1USsKwGf1728FFn1qAqVvjmZ
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

Required Cloud settings: `KOI_STRIPE_SECRET_KEY`, `KOI_STRIPE_WEBHOOK_SECRET`,
`KOI_STRIPE_PRICE_ID`, `KOI_PUBLIC_URL`. The default self-hosted mode ignores them.
Cloud mode also needs a Koi-operated `KOI_CLORO_API_KEY` to collect answers.

The account must be activated separately for live payments, with Stripe's business
verification, payout and tax settings completed in its Dashboard. Use live API keys,
a live Price and a live webhook signing secret only after sandbox flows pass. Sandbox
IDs and secrets are not interchangeable with live IDs and secrets.

## Data flow

Koi creates one Stripe Customer per organization and stores its ID before Checkout.
It creates a **hosted** Checkout Session with `customer`,
`client_reference_id=organization_id`, one Price and the selected prompt quantity.
The browser receives only the Stripe-hosted redirect URL. It never constructs a
Checkout URL or supplies a price or entitlement. A database reservation serializes
Checkout requests for the organization; retries reuse the same session.

`checkout.session.completed` records the subscription ID only when the signed event's
session ID, customer ID and organization ID match the reserved session and organization.
Subscription events are reconciled only against that completed Checkout and the
organization's stored Customer. Each event ID is recorded in the same transaction as
its state change, so replay is harmless. Premature subscription events are ignored;
the Checkout event fetches the current subscription to complete synchronization.
Email is never used as an identity key.

The database holds the application entitlement. `active` subscriptions grant their
selected allowance until the current period ends. `trialing` grants five prompts
until trial end, with a payment method collected in Checkout. Other statuses grant zero;
data is preserved after payment failure or cancellation. An immediate upgrade uses
Stripe proration and a pending update, so the larger entitlement arrives only after
Stripe applies the paid change. A downgrade is scheduled for the next cycle. If the
new limit is below the active prompt count, tracking stops until enough prompts are
paused. All prompts remain stored. Cancellation via Portal retains access until the
period end when Stripe reports `cancel_at_period_end` and `active`.

## Internal collection cost

Each visibility result records the engine, collector, submission time and, when
Cloro supplies `X-Credits-Charged` on the completed task response, the credits charged.
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

- Quote and Checkout for all 12 allowances match the same tax-inclusive Stripe amounts. Other
  quantities, including 6 and 501, are rejected.
- Two browser tabs requesting Checkout for one organization reuse one session.
- The webhook ignores a replay and rejects a Customer or organization mismatch.
- Upgrade gains entitlement only after Stripe confirms payment; downgrade changes it
  at renewal. Cancellation keeps access until the paid period ends.
- Paused prompts do not consume the allowance and already queued cells do not submit
  to Cloro after deactivation.
- A self-hosted install starts and tracks with its own Cloro key and no Stripe keys.

Checkout was visually checked in the sandbox at five prompts with an Italian
billing address: €9 after trial, including €1.62 IVA. The invoice preview was
checked at the same quantity and total; it still splits the zero-priced unit
and €9 flat-fee lines. Pending upgrades, schedule changes and Portal
cancellation behavior still require sandbox validation before production use.
