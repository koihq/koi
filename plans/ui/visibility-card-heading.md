# Visibility card heading

Status: Ready and implemented on the exploration. Baseline: `5046405` with existing uncommitted prompt-table and chart-preview work; preserve those edits.

## Target and research

The founder reads the main visibility card to understand mention rate, its trend, and competitors. The card already has a metric, chart, engine filters, and date range, but lacks a visible heading. The user supplied Fey's date-over-headline reference. Mobbin searches for a date/headline above a chart inspected [Fey's market card](https://mobbin.com/screens/f695556c-157e-445e-9975-c3f4e95f6b0d), which visibly uses a quiet date then short headline before the chart, and [Square's analytics page](https://mobbin.com/screens/73e78acc-e251-4502-b653-28e14fb40ccb), where an update timestamp stays secondary to metric content. Another [Fey portfolio screen](https://mobbin.com/screens/f4b1fb8e-1db1-407c-a1b1-41ec1eb312c7) has a chart but no date-over-headline match. The images establish visual hierarchy, not keyboard behavior or exact dimensions.

## Principles and checks

- Identify the metric immediately: show `AI visibility` above the percentage; retain the percentage as the largest type and the chart as the detailed evidence.
- Date the measured data: format the final day in `dailySeries` as `Tuesday, October 13` rather than using the wall clock. Keep the existing inspected-point date near the metric.
- Keep one reading group: date and title share the metric's leading edge; engine filters remain in the same header at the trailing edge. On narrow widths, the header may wrap without clipping.

| Skill/rule | Decision | Verification |
| --- | --- | --- |
| Product design and typography | Use existing type and color tokens; no new surface or decorative divider | CSS inspection; rendered hierarchy pending |
| Accessibility and layout | Make the section's existing `aria-labelledby` point to a visible `h2`; use native `time`; preserve DOM reading order | Type check; browser screen-reader and zoom check pending |
| Writing | Short sentence-case heading and an exact collected date | Inspect fixture date and rendered copy |
| Codebase patterns and ponytail | Keep this in the route-private component and its locale/CSS; no new dependency or shared abstraction | Diff inspection |

## Implementation handoff

1. In `home-signal.tsx`, format the last `series` date with weekday/month/day in UTC, render it in a `time` above a visible `h2`, and group those with the existing metric. Keep engine filters as the trailing header content.
2. In `prompt-visibility.en.json`, add `visibilityTitle: "AI visibility"`; keep `heading` for the metric-details tooltip.
3. In `-home-context.css`, use `--hc-type-meta` and `--hc-secondary` for the date, `--hc-type-title` and `--hc-text` for the heading, and a 16px group gap before the metric. Let the header wrap at constrained widths.

Acceptance: date/title/metric appear in that order; the title stays visible in light/dark modes; engine toggles, chart inspection, empty state, and competitor comparison keep their current behavior. Run `pnpm --dir ui check`, `pnpm --dir ui check-types`, and `git diff --check`. Browser appearance and 200% zoom still need a running Vite preview.
