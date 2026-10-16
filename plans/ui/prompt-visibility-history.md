# Prompt visibility history

## User job and scope

For one prompt, understand how often the brand appears over time, compare that trend
with one competitor, audit each collection run across engines, and inspect the exact
answer and sources from any engine. The page should give the graph and run evidence
the space currently spent on a narrow history list and inline reader.

Preserve the recorded Cloro examples already added for the zsh autosuggestion prompt.
Keep this within the existing exploration route and fixture data. No API, database,
dependency, or route changes are needed.

## Research

Mobbin screen research found useful patterns:

- [Peec AI overview](https://mobbin.com/screens/ea726eca-9ea2-423c-be52-181b642a04f1)
  pairs a trend with ranked comparison information.
- [Profound](https://mobbin.com/screens/4def7e6a-f280-4cf7-bfe8-e0c53429aef5)
  places competitor controls near its trend chart.
- [Mixpanel](https://mobbin.com/screens/fa5b6186-fa90-4744-a4e6-fd6bc73717b1)
  uses a detailed table beneath a graph for audit and exploration.

The second Mobbin query for run-detail screens returned metadata that did not match
its preview images, so those results were excluded. The user supplied Fey references
for readable answer text and a source rail; use that visual direction with Koi's
existing surfaces and typography.

## Design principles

1. Make the prompt's visibility trend the primary relationship. Use the existing
   `SignalChart` and 7/30/90-day control; show the prompt brand and at most one
   selected competitor, following the main page's compare behavior.
2. Make the run table auditable at a glance: date/time where known, named engine
   outcome pills with semantic color and tooltip text, and unique competitors found
   across the run's engines.
3. Keep answer inspection in a large modal that preserves the selected run and engine
   in the URL state. Show run time and summary, engine selectors, readable answer text,
   and sources in a right rail on wide screens. Match the Fey reading hierarchy: the
   prompt is the bold title, while run time and collection metadata stay compact and
   regular; answer prose is larger regular text.
4. Reuse Koi's components, existing color tokens, and evidence copy. Stack content on
   narrow screens and preserve keyboard selection, modal focus, escape-to-close, and
   descriptive labels.

## Existing behavior and data flow

- `PromptDetail` receives the prompt row, evidence records, brand and URL search state.
- `promptRuns(records, promptId)` groups results by `runId` and engine.
- `dailySeries(records, brand, scope, target)` yields gap-aware chart points. Use
  `comparedBrands` and `summarize` to form the competitor list; toggle one
  `compareId` through `onSearch`, matching `HomeSignal`.
- The run table aggregates engine results from `PromptRun.results`; competitors are
  deduplicated from each result's `appearances`.
- Selecting a date opens the run dialog. Selecting an engine pill opens that run with
  the engine preselected. `runId` and `responseEngine` already exist in route search.
  Closing clears `runId`; invalid ids show the existing missing-run recovery.
- Recorded evidence includes actual run timestamps; synthetic evidence remains date
  only so the UI does not imply a precise collection time that is not available.

The implementation follows the existing call shapes:

```ts
const scope = { days, promptIds: [row.prompt.id] };
const ownSeries = dailySeries(records, brand, scope, you);
const otherSeries = selected
  ? dailySeries(records, brand, scope, selected)
  : undefined;

onCompare(selected?.id === competitor.id ? undefined : competitor.id);
onSearch({ runId: run.id, responseEngine: engine.id });
```

`PromptRun` gains only an optional display timestamp. This stays compatible with
date-only synthetic fixtures and the existing API generated types; no persisted data
or API contract changes.

## View specification

```text
Prompt title                                             Edit

Visibility over time                     Compare with
┌──────────────────────────────────────┐ ┌─────────────────────────────┐
│ 30 days · 7 days · 90 days            │ │ ○ Competitor name  42%      │
│                                       │ │ ● Competitor name  38%      │
│       brand trend / competitor trend  │ │                             │
└──────────────────────────────────────┘ └─────────────────────────────┘

Run history
┌ Date and time ─ Engine results ────────────── Competitors found ──────┐
│ Oct 9, 2026   [ChatGPT · mentioned] [Perplexity · failed] Foo, Bar    │
│ Oct 8, 2026   [ChatGPT · cited]     [Gemini · missed]    Baz          │
└───────────────────────────────────────────────────────────────────────┘

Click run → large dialog: run date/time, rollup, engine buttons,
answer text at left, source list at right; stack source list on narrow screens.
```

Engine statuses use visible text and semantic color (mentioned, cited, missed,
failed, pending, or not collected). Tooltips explain each status. Each status button
opens the run dialog with that engine selected.

## Files and implementation sequence

1. Update `-components/prompt-detail.tsx`: replace the miniature strip and split
   history/reader with a two-column chart/competitor area and full-width run table;
   add the controlled large answer dialog. Reuse `SignalChart`, `BrandPill`,
   `AnswerBody`, and `SourceList`.
2. Extend `-lib/prompt-visibility.ts` with an optional run timestamp and deduplicated
   run competitor helper if it clarifies rendering. Add the real recorded timestamps
   in `-lib/home-evidence.ts`; keep synthetic run timestamps unspecified.
3. Update route-local CSS in `-home-context.css` for the graph composition, status
   pills, responsive table, and modal answer/source layout. Keep existing focus and
   reduced-motion conventions.
4. Add/adjust prompt-visibility tests for run time and competitor aggregation, and
   update `-locales/prompt-visibility.en.json` for all user-facing labels and statuses.

## States and failure cases

- No runs: retain the existing empty state.
- Outcome statuses distinguish cited, mentioned, missed, failed, pending, and not
  collected. Missing engine result is labeled unavailable.
- A run with an error still opens; the answer panel explains the failure and its
  sources are marked unavailable.
- Competitors without measured observations remain visible but cannot be selected;
  explain why in accessible text/tooltip.
- An unknown run id shows the existing recovery action to the latest run.
- Keep table horizontal overflow contained on small screens; status buttons remain
  independently operable.

## Acceptance checks

- `pnpm --dir ui check`
- `pnpm --dir ui check-types`
- Focused Vitest for prompt visibility and fixture evidence.
- `pnpm --dir ui build`
- `git diff --check`

Manual browser checks are left to the user per the repository's Fey visual-system
guidance: compare and clear a competitor; inspect chart periods; open a run by date
and directly by engine status; switch engines; follow source links; close with Escape
and confirm focus restoration; check narrow mobile widths, 200% zoom, and light/dark
themes.

## Skill preflight

Applied product-design, codebase-patterns frontend, ponytail, frontend-design,
Vercel React best practices, and the interface-review instructions for the final
diff review. The Mobbin research above was completed before implementation. The
implementation should retain the current Fey direction and use the existing Koi
component patterns rather than introducing a new visual system.

Status: ready for implementation (approved by the user's “go”).
