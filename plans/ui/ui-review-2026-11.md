# UI review — November 2026

## Scope

Authenticated product surfaces on branch `fix/ui-items`, rendered from `make dev` on
`:5173` with the demo seed (`Koi · Storico` for history, `Koi · Vuoto` for the empty state):

- Home (`/`): signal, competitors, prompt manager, empty state
- Prompt detail (`/prompts/$promptId`): chart, Compare with, run history, **run dialog**
  (answer per engine, brands in the answer, sources)
- Competitors (`/competitors`): table, search, add dialog, row menu
- Settings brand / connections, Sign-in: layout audit only

Each surface was checked at 1440 and 390px, light and dark. **Not reviewed:** Setup,
billing, OAuth consent, keyboard-only, 200% zoom, reduced motion, loading and error states.

Project sources read: `AGENTS.md`, `docs/design-system.md`, `product-design` (+ `primitives.md`),
`codebase-patterns/frontend.md`.

References: Fey home/watchlist (Mobbin), Perplexity Finance and Base for entity detail.
Trade Republic and Family are not on Mobbin web; their qualities come from
`product-design` → References.

## Method

- **Rendered screenshots** captured through the stageshow `layout-core` audit, run by an
  authenticated wrapper outside the repo.
- **Layout jumps** recorded with stageshow `jumps-core` while changing scope:
  - Home: range 7D → 90D and the compare competitor.
  - Detail: range and Compare with.
  - Competitors: search.
- **Hover captures** on the prompt table, run history and competitors table.
- **Stageshow lens** questions applied to each viewport.
- **Source read** of the cited files.

`film.mjs` and `jumps.mjs` require `.ss-stage` elements and cannot run on app pages, so
`layout-core` and `jumps-core` were reused directly.

Automated layout findings: the two toolbar `Select` triggers on Home and Competitors are 28px
high but their content needs 33.5px. The run dialog and the add-competitor dialog are clean.

## Findings

| # | Sev | Domain | Finding | Evidence | Fix |
|---|---|---|---|---|---|
| 1 | HIGH | ui / writing | **The run dialog does not show what the founder came for**, though it is the audit layer and the core of the product. The answer text does not mark the brand or the competitors. The outcome "Cited / Mentioned" does not say whose. "Brands in this answer" lists competitors only (Koi is cited but missing) and repeats the same evidence ("Named in answer: Peec AI / Named entity: Peec AI"). Sources are raw underlined URLs, with no favicon, no domain, and no mark for the brand's own domain. There is no agent action where the evidence lives. | `prompts/$promptId.tsx:487–540`, `Answer` `:556–652`; `extra/run-chatgpt-1440-dark.png` | Highlight the brand and competitor names in the answer, with the brand in `foreground` and rivals in `muted`, as quiet underlines rather than colored chips. Outcome as "Koi cited". One row per brand with its evidence kinds de-duplicated into small labels. Sources as domain + favicon, with own-domain sources first. "Copy for your agent" in the dialog header. |
| 2 | HIGH | layout / ui | **Values swap instead of moving.** On range change, the competitor lists re-sort in one frame: rows jump ±44/±88px on Home and ±59/±118px in detail Compare with, and the selected rival jumps with them. On Home, the engine filter and title shift 5.4px because the headline width changes. The chart height jumps 8.8px on 90D. | jumps-core: `"Scrunch" y -88px`, `"Peec AI" y +118.3px`, `group/button x +5.4px`, `recharts-surface height -8.8px` · `index.tsx:120`, `:315` | Keep row order stable within a session, or animate the reorder (FLIP, `transform`). Give the headline a fixed `ch` min width with `tabular-nums`. Pin the chart wrapper height. |
| 3 | HIGH | layout | **The chart wastes the signal.** The fixed 0–100 domain puts a 52% line flat across the middle, leaving about 120px of empty band above and below. In detail, a 100% line hugs the top over about 200px of void. | `_-1440-*`, detail screenshots · `components/blocks/visibility-chart.tsx:246` | Quiet 0/50/100 guides, or an adaptive domain with a visible floor. Height of about 180px. |
| 4 | HIGH | layout | **Two focal relationships in one surface on Home.** Headline, engines, chart, range and the competitor table share one `section`. Rival numbers sit about 1100px from their names at 1440. | `index.tsx:162–455`, `_-1440-dark.png` | Fey layout: signal on the left (≈2/3) and "Recommended instead" on the right (≈1/3) from `xl`, stacked below that. Numbers sit beside names. Detail already uses this split and is the model. |
| 5 | MEDIUM | layout | **On mobile, controls hide behind horizontal scroll.** Competitors at 390 shows only Name/Domain, so row actions sit off-screen behind a "Scroll horizontally" hint. Run-history engine chips are cut and the competitor column is off-screen. The prompt list goes the other way: 380px cards, a 7800px page for 20 prompts. | `extra/competitors-390-light.png`, detail `-390-light.png`, `_-390-light.png` | One mobile row pattern for all lists: primary label + one number + delta, with the action menu always visible and detail on tap. |
| 6 | MEDIUM | typography | **Parallel type scale.** `text-[13px]` ×19, `text-[30px]` ×3, `text-[36px]` ×2, and `font-bold` ×6 next to `font-semibold`. "AI visibility" (`text-xl font-bold`) matches no role in `primitives.md`. | `index.tsx:180`; grep | Token roles in `@theme` and a single strong weight. Swap to Hanken Grotesk at the same time. |
| 7 | MEDIUM | ui | **The row highlight differs by table.** The prompt table has the rounded whole-row fill (Trade Republic-like) and dims siblings to 0.55. Competitors and run history use shadcn `border-b` hairlines with a square `muted/50` fill. Prompt-row icons gain circular backplates only on hover. | `styles.css` `.prompt-table`; `components/ui/table.tsx:80`; `hover-prompts-dark.png`, `extra/hover-runs.png`, `extra/hover-competitors.png` | One `DataTable` `interactive` variant: no hairlines, rounded fill, sibling dim kept at 0.55, `background-color` 120ms, identical icon chrome at rest and on hover. |
| 8 | MEDIUM | colors / ui | **Engine identity is inconsistent.** Home uses monochrome Hugeicons glyphs. Detail and the run dialog use favicons through `BrandPill`, so Gemini and AI Overviews share the multicolor Google "G". The dialog's engine tabs put a pill inside a bordered box (nested chrome). | `prompts/$promptId.tsx:423`, `:516–521`; `index.tsx:56` | One monochrome `EngineIcon` block. The engine switch becomes a segmented control. |
| 9 | MEDIUM | ui | **Rows carry too much repeated data.** Prompt rows show four engine glyphs, a 30-segment barcode bar, a "points" sub-line and a near-constant 50.0% brand rate, about 8 strings per row. Run history repeats the same five competitor chips on every row, 3 lines tall. | `prompt-manager.tsx:275`, `:287`; detail screenshot | Prompt row: text, rate with a thin solid bar, delta chip, top rival. Run row: date, four outcome dots, rival count with 2 avatars. Details go to hover preview or the dialog. |
| 10 | MEDIUM | writing | **Numbers and dates lack units and agree poorly.** Dates appear as "Oct 4 – Nov 2", "10/4/2026" and "11/2/2026, 8:22:54 AM". Compare with shows "65.7 lower visibility" with no unit, while Home says "pp". The endpoint pill (25.0%) and the list (32.9%) show different numbers for the same rival without a label. "180 total" has no noun. | `prompts/$promptId.tsx:266`, `:269`, `:398`, `:500`; `extra/detail-compare-dark.png` | Shared `formatDay` / `formatDateTime` in `lib/`. "−65.7 pp vs you". Label the period average against the latest value. "180 answers". |
| 11 | MEDIUM | layout | **The prompt toolbar is overloaded.** It holds search, a status select, a sort select, an asc/desc toggle, a "Draft prompts and competitors" button and two separate error `<p>`s. The selects also fail the height check. | `prompt-manager.tsx:452–505`; layout audit | Search + segmented status + one Sort menu. Drafting moves into the header next to "Add prompt". Single error slot. |
| 12 | MEDIUM | writing | **The empty state stacks four explanations before an action**, and "No data –" renders as a headline. | `before/empty/_-1440-dark.png` | One sentence + one primary action ("Review prompts"). Schedule and credits go behind a disclosure. No headline metric without data. |
| 13 | MEDIUM | colors | **Theme parity breaks in the run dialog.** In dark, the answer and sources sit on distinct `surface` panels. In light, `surface` equals the dialog background, so both panels disappear and the hierarchy changes. | `extra/run-perplexity-bottom-1440-light.png` vs `run-chatgpt-1440-dark.png` | Inside dialogs, panels use `background` (canvas) or no panel at all, the same in both themes. |
| 14 | LOW | writing | **Competitors spends a column on "Added manually · Confirmed"** (identical on every row), shows no visibility, and offers "Start a first draft on Prompts" as a stray link. | `competitors/index.tsx:88` | Columns: name, domain, visibility, gap vs you. Status only as a quiet "Unconfirmed" chip. |
| 15 | LOW | ui | **Ad-hoc styling**: `-top-[7px]`, `-me-[7px]`, breakpoints `min-[1101px]`, `min-[441px]` and `max-[359px]`, and the glass recipe duplicated inline with `rounded-[12px]`. | `index.tsx:265`, `:426`; `prompt-preview-link.tsx:71`, `:87`; `prompt-manager.tsx:527` | Baseline alignment, a `-space-x-*` stack, container queries, `@utility surface-float`. |

Not reported: the Competitors search collapses the table in one frame (−245px). That is
expected for filtering.

## Stageshow lens: summary

| Question | Home | Detail | Run dialog | Competitors |
|---|---|---|---|---|
| Eye lands on the result? | Partly: flat chart, distant rival numbers | Yes (100%), then chip noise | No: the answer is plain text with nothing marked | No metric at all |
| Real strings in first viewport | ≈ 14 | ≈ 20 | ≈ 30 | ≈ 25 (repeated status) |
| One leading surface? | No | Yes (chart) | Yes in dark, none in light | Yes |
| Empty bands / dead corners | Chart bands, competitor gap | Under the chart line | Empty sources column below 3 links | Lower half of the page |
| Stock AI marks | Gemini sparkle (engine glyph, acceptable) | Multicolor Google G | Multicolor Google G ×2 | None |
| Swap vs move | Re-sort and width shifts | Re-sort in Compare with | — | Filter collapse (expected) |

## Verdict

**Needs changes.** Fix foundations first (#6, #7, #8, #15: tokens and shared blocks). Then
the audit layer (#1, #13), which is the product's core. Then the Home composition (#2–#4,
#9, #11, #12), then mobile lists (#5), then detail and Competitors copy (#10, #14).

## Resolution (Phase 2)

All 15 findings were addressed in the working tree. Verified with `pnpm check`, `check-types`,
`vitest run` (63 tests), `pnpm build`, `make go-format-check`, the stageshow `layout-core`
audit on Home, Competitors, Prompt detail, Settings brand and Settings connections at 1440/390
in light and dark (0 findings, previously 2 per page on Home and Competitors), and `jumps-core`
on range and compare changes.

| # | Resolution |
|---|---|
| 1 | `prompts/-components/run-evidence.tsx`: brand outcome in words, brand and rivals underlined in the answer (`lib/answer-marks.ts`), de-duplicated evidence kinds, sources by site with own site first, "Copy for your agent" (`lib/agent-brief.ts`) |
| 2 | `hooks/use-flip.ts` on Home and detail rival lists. No one-frame re-sort jumps remain. The remaining chart shift is the documented scope-change feedback |
| 3 | Rejected by the product owner: the chart stays clean in the Family style (no guides, 220px). Guides were tried and reverted |
| 4 | Home: signal + "Recommended instead" side by side from `xl` |
| 5 | Prompt and competitor lists switch to stacked rows in narrow containers (`@container`); run history is a row list. The mobile Home is 4530px, down from 7800 |
| 6 | Hanken Grotesk + `text-body/section/title/metric` tokens; `font-bold` removed |
| 7 | `DataTable interactive` + `.list-rows`; icon backplates removed |
| 8 | `components/blocks/engine-icon.tsx`, `outcome-mark.tsx`; segmented engine switch in the dialog |
| 9 | Prompt rows: text, rate + bar + delta chip, top rival (engine and rival detail on hover). Run rows: date, outcome marks, rival icons |
| 10 | `lib/format.ts`; "−65.7 pp vs you", "30-day average", "180 runs" |
| 11 | Toolbar: search, status segments, Sort menu; drafting in the header; single error slot; Select primitive height fixed |
| 12 | Empty Home: one sentence + "Manage prompts"; cadence moved into the metric info; empty pagination hidden |
| 13 | Dialog panels removed; same hierarchy in both themes |
| 14 | Competitors: visibility and gap columns, status chip only when unconfirmed, stray draft link removed |
| 15 | `surface-float` utility; offsets and arbitrary breakpoints replaced; legacy Sidebar, Sheet and `use-mobile` removed |

Not verified: keyboard-only pass, 200% zoom, reduced motion, Safari and touch devices, and Go
tests (`make test`; no Go files changed).
