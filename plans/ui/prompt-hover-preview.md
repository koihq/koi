# Prompt hover preview

**Status:** Ready. **Target:** prompt links in `/dev/explorations/home-context` table. **Baseline:** `5046405` plus the current uncommitted prompt table redesign. This is an exploration using fixed synthetic evidence; it adds no API request or persisted state.

## User job and research

The founder wants to decide which prompt to inspect without opening each detail screen. The preview summarizes the same scoped mention evidence that the link opens. Mobbin `search_screens` examined [Mobbin's gallery](https://mobbin.com/screens/78c8c297-b03a-4030-a8f2-25ab2e9bd133), [Gumloop's compact analytics charts](https://mobbin.com/screens/5a669967-687f-408b-9030-51dde29ea024), and a finance table preview query, which returned no useful hover preview. The user's two screenshots are the decisive references: an enlarged adjacent preview and a thin line over a translucent filled area. Static Mobbin results do not establish tooltip timing or keyboard behavior. Koi's existing `SignalChart` sets the line weight, and the existing brand switcher sets the overlapping logo geometry.

## Principles and preflight

Preview content follows the question's decision order: your mention rate and top competitors, then the trend for the selected period and engines. It should be supplemental, not another interactive screen, so it opens from the prompt link on mouse hover or keyboard focus; the same link still navigates on click, Enter, or touch. No row-level trigger: checkboxes, action menus, and engine breakdowns must remain independent. Keep a 320px floating surface, use the existing tooltip primitive, `BrandPill`, `dailySeries`, and `trendPath`, and fill beneath the 4px line with a neutral gradient. Do not invent data for drafts or unmeasured prompts. The `TooltipContent` remains noninteractive, and there is no new motion beyond the existing tooltip behavior.

| Skill/rule | Decision | Verification |
| --- | --- | --- |
| product-design/codebase-patterns | Scope to the route-private `PromptList`; no shared primitive or API change | Existing detail link works |
| better-accessibility | Link remains semantic; focus opens tooltip, Escape closes, touch navigates | Keyboard and touch |
| better-layout/better-ui | Collision-aware portal tooltip, bounded width and stacked brand logos | Both themes and viewport edges |
| better-colors | Existing surface and text tokens; translucent fill uses chart line token | Rendered contrast |
| animate/ponytail | Reuse Tooltip; no extra hover animation or dependency | Rapid row hovers |

## Contract and implementation

`Workspace.records → Management.records → PromptList.records → PromptPreview` supplies the existing synthetic `EvidenceRecord[]`. `dailySeries(records, brand, {days: search.days ?? 30, engines: search.engines, promptIds: [row.prompt.id]}, measuredBrand)` supplies chart points. The tooltip renders only when open. The competitor stack is `row.topBrands` without the current brand, limited to three; zero competitors gets an explicit empty label. The chart uses a 4px non-scaling line matching `.hc-trend-line` and a linear gradient beneath each contiguous run. Empty series shows “No answers yet” instead of a fabricated graph.

Refinement from rendered feedback: the tooltip is 290px wide, the chart view box is 320×96, and the area retains 8% opacity at its baseline. The chart bleeds to the tooltip's inner left, right, and bottom edges; rounded clipping contains it. Remove the visible period footer. The graph's accessible name still states the active period and start/end rates.

1. Pass `records` from `Workspace` through `Management` to `PromptList` in `home-context.tsx`.
2. In `prompt-list.tsx`, wrap only the prompt link in the existing Tooltip. Add a focused preview component with rate, competitor icon stack, and a lazily computed scoped chart. Preserve TanStack Link navigation and `onOpen` scroll restoration.
3. Add `trendAreaPath(points, baseline)` in `-lib/trend-path.ts` using the same monotone segments as `trendPath`; preserve gaps. Test empty, single-point, and separated runs in its existing test file.
4. Add scoped tooltip, brand stack, and graph styles in `-home-context.css`; hide the primitive arrow like other Koi tooltips. Use existing locale copy where possible and add exact missing labels to `prompt-visibility.en.json`.

Preview state is derived from current rows and search scope, so filtering, brand switching, period changes, and pagination replace it naturally. No loading or recoverable network error applies to this fixed exploration. Open on mouse hover and focus-visible, close on leave, blur, Escape, click, or unmount. No touch preview. Tooltip collision positioning should keep it within a 320px viewport.

Verify `pnpm --dir ui check`, `pnpm --dir ui check-types`, `pnpm --dir ui test`, `git diff --check`. In Vite at `http://localhost:5173`, inspect first, middle, and last table rows in both themes; hover near right/bottom viewport edges; Tab to a prompt and press Escape/Enter; touch a prompt; inspect a draft with no evidence; and compare preview values and trend shape against the opened detail with the same period and engine filters. Runtime verification remains required if no browser is available.
