# Koi Cloud billing (local implementation)

Koi Cloud bills one organization for a monthly allowance of active tracked prompts. All
features, brands, members and supported engines are included. Self-hosted installations
use `KOI_DEPLOYMENT_MODE=selfhosted` (the default) and never call Stripe. They still
need their own Cloro key for live collection.

## Pricing

`pkg/koipricing/catalog.json` contains **interpolation anchors**, not selectable bands.
Every whole prompt count from 5 through 1000 is valid. For example, 6 prompts cost
€10/month. The server interpolates in integer cents, rounds to the nearest whole euro
(half up), and rejects quantities outside the range. The Stripe CLI commands below
create one licensed, monthly, volume-tiered Price; they group
equal totals into 540 tiers. This is one Price per catalog version, not hundreds of
separate Prices or subscriptions. Stripe is queried for each quote and its tier totals
and version are checked against the local catalog before Checkout.

The final Stripe tier uses `up_to=inf` because Stripe supports a fallback tier. Koi's
API still enforces the 1000 maximum. Do not enable customer-adjustable quantities in
Checkout or Portal. Updating the catalog requires a **new version and Price ID**;
existing subscriptions retain their original Stripe Price.

## Sandbox setup

Do these steps in a Stripe **sandbox** before enabling Cloud mode. The local code does
not create or modify objects in the account automatically at startup.

1. Log in with the installed CLI: `stripe login`.
2. Create one Koi Cloud Product and copy its `prod_...` ID:

   ```sh
   stripe products create --name "Koi Cloud"
   ```

3. Set `KOI_STRIPE_PRODUCT_ID=prod_...` in your shell. From the repository root,
   run this one-time terminal command. It reads the versioned anchor catalog,
   groups equal monthly totals, and invokes the Stripe CLI to create one Price.
   There is no account provisioning command in the Koi binary.

   ```sh
   export KOI_STRIPE_PRODUCT_ID=prod_...
   python3 - <<'PY'
   import json
   import os
   import subprocess

   with open("pkg/koipricing/catalog.json", encoding="utf-8") as source:
       catalog = json.load(source)
   anchors = catalog["anchors"]
   prices = {}
   for low, high in zip(anchors, anchors[1:]):
       width = high["prompts"] - low["prompts"]
       rise = high["euros"] - low["euros"]
       for prompts in range(low["prompts"], high["prompts"] + 1):
           offset = prompts - low["prompts"]
           prices[prompts] = 100 * (low["euros"] + (2 * rise * offset + width) // (2 * width))

   tiers = []
   for prompts, cents in sorted(prices.items()):
       if tiers and tiers[-1][1] == cents:
           tiers[-1] = (prompts, cents)
       else:
           tiers.append((prompts, cents))
   assert len(tiers) == 540 and prices[6] == 1000 and prices[1000] == 54900

   command = [
       "stripe", "prices", "create",
       "-d", f"product={os.environ['KOI_STRIPE_PRODUCT_ID']}",
       "-d", f"currency={catalog['currency']}",
       "-d", "recurring[interval]=month",
       "-d", "recurring[usage_type]=licensed",
       "-d", "billing_scheme=tiered",
       "-d", "tiers_mode=volume",
       "-d", f"metadata[pricing_version]={catalog['version']}",
       "-d", f"nickname={catalog['version']}",
   ]
   for index, (up_to, cents) in enumerate(tiers):
       command += ["-d", f"tiers[{index}][up_to]={'inf' if index == len(tiers) - 1 else up_to}"]
       command += ["-d", f"tiers[{index}][flat_amount]={cents}"]
   subprocess.run(command, check=True)
   PY
   ```

   Copy the returned `price_...` as `KOI_STRIPE_PRICE_ID`. The application
   checks all 996 selectable quantities against Stripe before quoting or Checkout.
   If Stripe rejects the tier count, stop here and revisit the Price model before
   accepting purchases.
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
`client_reference_id=organization_id`, one Price and the exact prompt quantity.
The browser receives only the Stripe-hosted redirect URL. It never constructs a
Checkout URL or supplies a price or entitlement. A database reservation serializes
Checkout requests for the organization; retries reuse the same session.

`checkout.session.completed` records the subscription ID only when the signed event's
session ID, customer ID and organization ID match the reserved session and organization.
Subscription events are reconciled only against that completed Checkout and the
organization's stored Customer. Each event ID is recorded in the same transaction as
its state change, so replay is harmless. Unknown or prematurely delivered subscription
events return an error so Stripe retries them. Email is never used as an identity key.

The database holds the application entitlement. `active` subscriptions grant their
purchased quantity until the current period ends. `trialing` currently grants five
until trial end, but Koi does **not** start trials yet. Other statuses grant zero;
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

- Quote and Checkout for 5, **6**, 25 and 1000 prompts match the same Stripe amounts.
- Two browser tabs requesting Checkout for one organization reuse one session.
- The webhook ignores a replay and rejects a Customer or organization mismatch.
- Upgrade gains entitlement only after Stripe confirms payment; downgrade changes it
  at renewal. Cancellation keeps access until the paid period ends.
- Paused prompts do not consume the allowance and already queued cells do not submit
  to Cloro after deactivation.
- A self-hosted install starts and tracks with its own Cloro key and no Stripe keys.

The tier count, Checkout totals, pending upgrades, schedule changes and Portal
cancellation behavior still require sandbox validation before production use.
The Stripe CLI is not available in this local shell, so the provisioning command
above has not been executed or verified against the sandbox account.
