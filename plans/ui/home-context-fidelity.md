# Home-context fidelity pass

- **Status:** Ready
- **Target:** Authenticated Home, prompt detail, competitor management, and shared header.
- **Baseline:** `f9b0153` with the current uncommitted home-context migration. The exploration at `/dev/explorations/home-context` is the visual and interaction specification.
- **Scope:** Match that exploration on desktop and mobile while showing one brand and retaining the existing cookie-authenticated APIs. Aggregate rates remain deterministic mock data; saved run answers remain real.

## Research and direction

This revises the same UI plan documented in `docs/fey-visual-system.md` and `plans/ui/mvp-exploration-restart.md`; its inspected Mobbin research is reused. Fey portfolio (`https://mobbin.com/screens/f4b1fb8e-1db1-407c-a1b1-41ec1eb312c7`) showed compact aligned comparison rows. Fey home (`https://mobbin.com/screens/64282196-4e6c-4590-a0d1-07bde4e4819b`) showed a leading signal. Peec AI (`https://mobbin.com/screens/0711fee8-0de2-4f0e-8b7c-16387d22f091`) showed quiet evidence labels; its dashboard composition was rejected. The repository's rendered exploration resolves the remaining details more precisely than those external screens.

The founder first reads the overall rate and competitor gap, then finds which prompt and answer explain it. Keep the signal as the first full-width surface, with the heading, rate, graph, controls, and competitors inside it. Keep management as the second surface. Prompt detail keeps trend, comparison, run history, and answer evidence in that order. The single-brand exception changes only the switcher content, not its compact header geometry. No decorative motion is needed.

## Skill preflight

| Rule | Decision | Verification |
| --- | --- | --- |
| Product design / frontend design | Reproduce the exploration's one-signal composition, palette, type roles, and quiet management surface. | Compare desktop and mobile screenshots. |
| Codebase patterns / ponytail | Reuse the existing API wrappers, query keys, shadcn primitives, and exploration trend-path helper where appropriate. No dependency or backend change. | Types and tests. |
| Accessibility / layout / typography | Native controls, visible focus, complete prompt rows at 320px, no horizontal prompt-list scroll, full prompt text. | Keyboard and 320px browser pass. |
| Writing / colors / UI | Use the existing locale, status tokens, and home-context color values; explain empty data and recovery. | Both themes and error/empty states. |
| React practices | Reuse cached all-prompt and all-competitor queries; avoid per-row requests. | Inspect network and list mutations. |

## View contract

Desktop: compact icon-only brand header; full-width signal surface; 24px gap; full-width prompt surface. Signal: date/title and metric upper left, four icon engine controls upper right, smooth graph across surface width with endpoint label, date range and period pills beneath, competitor rows last. Prompts: heading/add action, compact search/status/sort/draft toolbar, evidence table and pagination. At 1100px and below, rows become stacked cards with every field visible; no table overflow. The period and engine filters share URL state. A confirmed prompt with no collected run still shows temporary mock analytics, while actual answer history is explicitly empty.

Prompt detail: back link and editable title, two-column trend/comparison region on desktop, single column on mobile, run-history table with per-engine statuses and found competitors, then a dialog for the selected run's answer and sources. Run and engine selection belong in URL search, so back/refresh preserve context. An absent prompt snapshot or failed history read gets a recoverable state. Competitors use the same surface and compact toolbar as prompts, including an identity action for observed names.

## Data flow

Home URL filters → `useQueryStates` → `mockSeries/mockRate` over confirmed records; prompt list and competitor list → existing `ApiProvider` wrappers. Prompt rows use one cached all-page prompt query and one cached all-page competitor query already loaded by Home. Filtering and sorting happen before slicing the requested page; unavailable rates stay last. Bulk confirm/delete invoke existing single-item mutations sequentially with error feedback and query invalidation. Prompt detail → existing `api.runs.promptHistory(brandID, promptID, page, 10)`; the dialog reads the selected run in that page and renders actual answer/source data.

## Ordered work

1. Update the authenticated shell and shared header to the exploration's compact geometry and local Calibre/Helvetica fallback. Retain the single brand and account actions.
2. Replace the Home signal's layout and graph treatment with the exploration's date/title, inline rate/change, icon filters, endpoint label, period pills, and quiet competitor rows. Keep current query and URL contracts.
3. Replace the prompt manager's generic table presentation with a matching compact surface, sort, full breakdown columns, past filter, selection actions, and stacked mobile rows. Reuse existing create/edit/delete/review/draft flows.
4. Recompose prompt detail to match the trend/comparison layout and run-history-to-dialog flow. Keep real answers and citations.
5. Recompose the competitor route with the same management surface and toolbar. Keep existing mutations and server pagination.
6. Update locale and design-system documentation, then run format, typecheck, tests, build, and rendered checks in light/dark at 1440px and 390px.

## Acceptance

- At 1440px, the first viewport has one signal surface and the next prompt surface, with no external page title or controls above the signal.
- At 390px and 320px, prompt rate, engines, brands, actions, and full prompt text remain available in the row without horizontal scrolling.
- Sorting operates over all matching prompts before pagination; bulk actions confirm or delete only selected records and report failures.
- A run row opens the actual saved answer and sources; Escape restores focus to its trigger.
- No new dependency or backend endpoint is required. `pnpm --dir ui check`, `check-types`, `test`, and `build` pass.
