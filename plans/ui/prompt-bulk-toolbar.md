# Floating prompt bulk toolbar

**Status:** Ready. **Target:** selection actions in `PromptList` on `/dev/explorations/home-context`. **Baseline:** `5046405` plus the uncommitted prompt table changes. The bulk action handlers, selection rules, and confirmation dialog already work; this changes placement and presentation.

## User job and research

Selecting a prompt should reveal actions without pushing the filters and table downward. Mobbin `search_screens` covered Fey financial tables, Linear issue lists, Shopify order tables, and a cross-product floating bulk toolbar query. [YNAB's transaction table](https://mobbin.com/screens/13e2610f-4a0c-4a66-8c5c-e07d1615eb61) visibly centers a compact selection bar above the viewport bottom. [ClickUp](https://mobbin.com/screens/e9639493-e0a6-46c9-93d1-d3189cbdc3c7) uses a much wider bar, so Koi should borrow its persistent access but not its density. The inspected [Fey](https://mobbin.com/screens/f9c3ac62-dc0d-4450-822f-85a9f9f9833b) and [Linear](https://mobbin.com/screens/fd1b4d88-f021-49a3-98af-4cd3a87e1d29) screens do not establish a bulk toolbar pattern. Exact measurements below are Koi choices, not values inferred from static images.

## Direction and preflight

Keep the existing selected count, Confirm, Delete, and Clear selection actions in one floating pill at viewport bottom center. It appears only while at least one prompt on the visible page is selected, including while scrolling. It uses Koi's existing floating surface tokens, not a new design system control. The table's existing bottom padding keeps content reachable. On narrow screens, the pill stays within the viewport and wraps actions if needed. Keep DOM order directly before the table so keyboard users encounter it after filters and before rows. Do not add entrance motion to this frequent action; appearing immediately is clear feedback.

Refinement after rendered feedback: show the selected count as plain text, Confirm as the primary filled action when present, Delete with the existing destructive variant, and Clear selection as a neutral ghost action. The toolbar says “Delete” because the count already gives its scope. Confirm says “Confirm” when all selected prompts are drafts; if only a subset is confirmable, retain “Confirm {count}” to make that smaller scope explicit. The delete confirmation dialog keeps its count.

| Skill/rule | Decision | Verification |
| --- | --- | --- |
| product-design and codebase-patterns | Scoped CSS and existing route-private `PromptList` | Dark/light surface fit |
| better-accessibility | Preserve real buttons, labels, dialog focus return, and tab order | Keyboard, zoom, mobile |
| better-layout and better-ui | Fixed centered pill, bounded width, safe area offset, existing shadow tokens | No collision or clipping |
| animate and ponytail | No new motion, dependency, or state | Rapid selection changes |

## Implementation handoff

1. In `ui/src/routes/dev/explorations/-components/prompt-list.tsx`, retain the existing conditional action group and handlers; add an accessible group label if helpful. No state or API change.
2. In `ui/src/routes/dev/explorations/-home-context.css`, position `.hc-prompt-selection-actions` fixed at `left: 50%`, `bottom: max(24px, env(safe-area-inset-bottom))`, with `translateX(-50%)`, z-index above the table, 12px radius, surface and floating shadow tokens. Use `max-width: calc(100vw - 32px)`, compact padding, and mobile wrapping. The bar must fit at 320px and not hide the last table row or pagination.

Initial and empty selection: no toolbar. Selected: count and valid actions visible. Confirm or Clear: selection resets and toolbar disappears; existing heading focus behavior remains. Delete: existing confirmation dialog opens and returns focus as before. No loading or error contracts change.

Verify `pnpm --dir ui check`, `pnpm --dir ui check-types`, `pnpm --dir ui test`, and `git diff --check`. In Vite at `http://localhost:5173`, inspect both themes, desktop and 320px width, 200% zoom, bottom scroll, keyboard selection and action focus, and delete dialog layering. Runtime inspection remains required where browser access is available.
