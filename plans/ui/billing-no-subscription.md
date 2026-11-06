# Billing without a subscription

- **Status:** Implemented, uncommitted, on `feat/add-a-judger-for-mentions`.
- **Target:** `ui/src/routes/_authenticated/_settings/settings/billing.tsx`, when the organization
  has no subscription record. No backend or OpenAPI change.

## Problem

The "Current subscription" panel took two thirds of the page to say there was none, while price
and action sat in the narrow column. First Look promises "€0 today", but this page led with the
full monthly price and the button did not mention the trial.

## Research

`search_screens`, web: billing page with no subscription; single plan with usage slider; Fey plan
settings; AI visibility plan by prompts; trial page showing today and the charge date.

| Reference | Observed | Decision |
| --- | --- | --- |
| [Churnkey](https://mobbin.com/screens/fb3f5f54-8d6d-4fdc-99ad-69bde8a7a683) | Volume, price and trial start in one block | Borrow |
| [Amplitude](https://mobbin.com/screens/ddb6e909-526f-418f-b998-21e2a3dd97f8) | Slider paired with "due today" | Borrow the pairing |
| [Semrush](https://mobbin.com/screens/3b447b96-3b49-4c3c-8bc8-dd5b1b884acf) | Today's charge $0.00 and a trial timeline | Borrow the today-then order; a sentence replaces the timeline |
| [Searchable](https://mobbin.com/screens/4752ea53-15bb-449b-8ccb-644d4ccf964a) | The free period leads, the later price follows | Borrow |
| [Fey](https://mobbin.com/screens/d74fefaf-87d7-4616-aa63-07c0ee8e0ada) | Centered column, a reminder before charging | Borrow the column. No reminder: Koi sends none |
| [Fibery](https://mobbin.com/screens/00bda83a-6ac2-4523-8aa4-13cb0eaae575) | Plan comparison grid | Reject: Koi has one plan |

Family and Trade Republic were not searched for this page.

## Decisions

- **One panel** when `subscription.isSuccess && !record`, outside a returning checkout and
  confirmation. Any subscription, canceled included, keeps the two-column page.
- **The price leads.** `€0 today` uses `text-metric`, the later price sits below it in
  `text-section`, and `TickPicker compact` makes the prompt count a control rather than a rival.
- **Default stays 15** (€29), matching the price First Look quotes.
- **Without a trial**, the monthly price leads and the action stays "Continue to checkout".

The 7 days and 10 prompts in the copy mirror `TrialPeriodDays` and `TrialPromptLimit`; the API does
not expose them.

## Verification

Done on a Vite server with a mocked API: trial at 1440, 800, 390 and 320px in both themes, no
trial, catalog failure, a failed checkout, the active and canceled pages, keyboard change of the
allowance updating price and copy. No horizontal overflow.

Not done: a real Stripe checkout, screen-reader and 200% zoom walks, measured contrast.
