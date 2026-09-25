# Koi roadmap

Koi is a developer- and agent-first AEO tool. When buyers ask ChatGPT, Perplexity, Gemini, or Google AI Overviews about your category, it shows whether you are mentioned or cited, **who is recommended instead**, which URL is winning, and a markdown brief your existing agent can execute — then it remeasures.

Time-to-value is the first visibility run, not a week of empty charts. Daily tracking exists so the loop stays closed.

**In:** a technical founder measuring their own product, living in Cursor / Claude / the terminal.

**Out:** agencies, GEO hire, white-label, an in-app writer, BYOK as the cloud SKU.

```
onboard → generate prompts + competitors → first run → inspect who-shows-up
        → gap brief → agent applies in your repo → koi check → daily remeasure
```

## Principles

- **No composite score.** Per cell: `cited` > `mentioned` > `miss` (plus `error`). Brand name in the answer; brand domain/URL in sources. Same scoring for each confirmed competitor. Raw answer + citation URLs are the record.
- **No writer.** Koi tells you what to fix. Your agent edits your repo. Koi does not open PRs or ship the page.
- **Dual surface.** Every v1 job exists in the browser and via CLI/MCP, on the same objects. Neither is a sidecar.
- **Engines included.** ChatGPT, Perplexity, Gemini, Google AI Overviews on every SKU. No engine add-on product. Claude stays out of operator-paid collect.
- **Human confirms before spend.** Generated prompts and competitors are curated before the first collect. Observed competitors from a run are suggestions until confirmed.
- **One brand, founder copy.** No client packs, demo-call onboarding, or agency dashboards.

## Now — MVP

Ship in this order. Self-hosted email/password cookie sessions precede Slice 1. Slice 2 starts as soon as slice 1 works. Slice 3 is the same MVP, not a later product.

### Slice 1 — See (AI visibility)

Onboarding and the first prompt × engine run. Includes competitor identification.

- [ ] Brand onboarding: name, domain or GitHub repo, country, optional one-line product
- [ ] OpenRouter onboarding agent (not the collect pipeline) drafts, in one pass:
  - ~10–15 buyer prompts: category, alternatives-to, comparison, use-case — not only vanity branded queries
  - ~5–8 likely competitors: `name`, `domain`
- [ ] Human review of both lists (accept / edit / drop / add) before any collect
- [ ] First run: persist prompt, engine, fetched_at, outcome, raw answer, sources, mentioned/cited entities
- [ ] Who-shows-up: attach rows to confirmed competitors; repeating unknowns become suggestions, not silent adds
- [ ] UI: live progress, per-engine breakdown, filters, answer detail, competitor panel
- [ ] CLI/MCP: `status` / `answers` / `competitors` on the same run

**Done when:** a new brand can go from empty onboarding to a completed matrix without already knowing which prompts to track or which competitors to name, then confirm the observed list.

### Slice 2 — Fix (close the loop)

Hand the next artifact to the founder’s existing agent. Remeasure.

- [ ] `citation_gaps`: URLs/hosts cited in losing cells that are not the buyer’s domain, grouped under a confirmed competitor when known
- [ ] `gap_brief`: markdown task with objective, mentioned/cited/miss counts, prompt table, who beat you and on which URL, evidence, recommended changes, acceptance criteria `koi check` can verify
- [ ] Apply path: MCP `gap_brief` + CLI `koi brief`. Agent edits the repo. No auto-PR.
- [ ] `koi check`: re-run the same brand + prompts; per-cell ↑ / ↓ / →

**Done when:** Cursor can pull a brief, change docs/README, and `koi check` shows the delta — dashboard never required for that path.

### Slice 3 — Stay (daily)

Replace the spreadsheet. Keep the loop closed.

- [ ] Daily, idempotent dated rows (prompt × engine × date)
- [ ] History in UI and `status`; gaps refresh from the latest day
- [ ] CSV/JSON export of the same log the UI shows
- [ ] Observed-competitor suggestions keep accruing; the confirmed set is what briefs and `koi check` use

**Done when:** yesterday and today are comparable without a one-shot rerun, and competitor drift is visible in the dated log.

### Surfaces (neither is a sidecar)

| Job | UI | CLI | MCP |
|---|---|---|---|
| Brand | onboarding form | `koi init` | `init_brand` |
| Prompts | generate + edit list | `koi prompts generate/add` | `generate_prompts` / `add_prompts` |
| Competitors | draft + observed + confirm | `koi competitors` | `list_competitors` / `confirm_competitor` |
| Visibility | run view + cell detail | `koi status` | `get_status` / `get_answers` |
| Gaps | gaps panel | `koi gaps --md` | `citation_gaps` |
| Brief | copy / download | `koi brief` | `gap_brief` |
| Recheck | re-run on a run | `koi check` | `check_run` |

Tracked competitor: `name`, `domain`, source (`drafted` | `observed` | `manual`), confirmed boolean. Prompt mix uses confirmed competitors for `alternatives to X` and `X vs Y`. Mention/cite detection uses the selected brand name and domain.

## Next

After the loop is dogfoodable.

- Operator-paid cloud (no user collect keys); self-host remains first-class
- Billing and a fetchable `/pricing.md`
- Agent-native packaging: README MCP snippet, `/llms.txt`, OpenAPI for the same routes MCP exposes
- Markdown comparison pages (`/vs/...`)
- Extra engines that have a different collect cost

## Not now

- In-app writer, GEO hire, auto-PR, “we ship the page”
- Composite 0–100 AEO or share-of-voice score as the product
- White-label, SSO, multi-brand client packs, shareable agency dashboards
- Claude included on operator keys; live scrape-through-MCP at tracker volume
- Public citation index
- Demo-call / sales onboarding
- Collect spend on generated prompts or competitors before the human confirms the lists
- Competitor battle cards or outreach to publishers
