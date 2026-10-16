# Daily chart inspection updates the headline

## Title and baseline

- **Status:** Ready as a design plan; implementation has not started, per the user's request to review the approach first.
- **Target:** `/dev/explorations/home-context`, `HomeSignal` and its `SignalChart`.
- **Baseline:** Commit `db6aa46`; relevant UI and product-design files are already modified in the working tree. The current chart has private pointer/touch/keyboard inspection state, an endpoint `BrandPill`, a tooltip with daily brand/comparison rates and sample counts, and a range/engine/competitor `scopeKey`.
- **Scope:** Show the inspected day in the existing top-left visibility metric, remove the chart tooltip, make the selected point and headline respond together, and keep the chart's final point inset from the right edge.
- **Constraints:** Keep this in the existing local exploration and fixture data. Preserve the dark/light exploration styles, current comparison curve, period controls, keyboard and touch inspection, and the no-extra-copy rule. No API, dependency, route, persistence, or production behavior changes.

## User job and current behavior

The founder scans the selected brand's mention rate, compares its trend with an alternative, and inspects a day. Today, moving across the line opens a tooltip over the chart while the headline continues to show the period aggregate and period-over-period change. The endpoint marker nearly touches the chart's right edge.

After the change, moving the inspection point updates the existing headline to that date's mention rate and its change from the previous date with completed answers. A compact date identifies the selected observation. The marker remains on the chart without a tooltip. Leaving the chart restores the period aggregate; touch selection remains until dismissal, matching the existing interaction.

## Research report

| Lane | Mobbin tool and query | Inspected result | Follow-up or limitation |
| --- | --- | --- | --- |
| Koi visual reference | `mobbin_search_screens`: “Fey web dashboard daily mention rate time series with selectable date range and brand filters” | [Fey stock detail](https://mobbin.com/screens/7e486036-e843-4576-a496-f8b1a5685f22) | Same compact headline/time-range/chart relationship, but it is a finance screen and its analysis controls are denser than this Koi surface. |
| Same interaction | `mobbin_search_screens`: “Family app Ethereum asset detail screen price trend chart and selected time period” | [Family Ethereum detail](https://mobbin.com/screens/640b5bd8-37e6-4aa6-86d6-8a96d91a8ae7) | The still shows a haloed chart endpoint, a headline value and a selected time range, with no tooltip. It does not establish motion or scrubbing behavior. |
| Other sector | `mobbin_search_screens`: “Stock portfolio chart scrubbing a selected point updates the large price and period change above the chart” | [Yahoo Finance chart](https://mobbin.com/screens/1430a6ec-a0d3-4bc6-93d2-215ca0e87c55) | It demonstrates crosshair inspection but adds axes and data overlays that are too dense for Koi. Borrow the connection between inspection and headline data only. |

The Family screenshot attached by the user and the Mobbin image for the same screen were inspected. The selected date and headline relationship is visible; the supplied material is a still image, so animation timing below is an implementation choice rather than an observed Family behavior. The selected project reference remains Fey's compact analytical hierarchy, adapted to Koi's mention-rate vocabulary.

## Principles and selected direction

1. **Inspection should update the decision metric in place.** The founder is already reading the headline as the chart's summary. Selecting a day therefore changes that same rate and delta, instead of opening a second floating data surface. The chart remains the spatial context; a compact date beside the metric identifies the value. This preserves the existing hierarchy and trades away the tooltip's extra sample-count row.
2. **Keep the comparison line visible and the primary brand metric singular.** Family's one-asset screen supports a single changing headline. Koi keeps its competitor curve and competitor summary below; the headline remains the active brand's daily rate. The `pp` change is the active brand's difference from its previous completed data day, as selected by the user. The comparator's exact daily rate remains available in the range input's accessible value text.
3. **Make chart state changes visible and brief.** A selected marker moving between days and the headline changing are state feedback. Animate only pointer-driven changes; keyboard selection updates immediately. Filter/range changes get a short plot transition. Reduced-motion users receive the updated values without transform motion.
4. **Give the endpoint room to breathe.** The chart domain ends 28px before its right edge. The endpoint badge remains attached to the last point and within the chart; no axis labels or extra right-side controls are added.

Alternatives rejected: keeping the tooltip would preserve redundant chart copy and still obscure the curve; adding a second headline for the competitor would turn a simple selected value into a comparison widget; using a day-over-day delta from the immediately preceding calendar date would produce missing deltas on days without completed samples.

## Skill preflight

| Skill and relevant rule | Decision checked | Adjustment or rationale | Verification still needed |
| --- | --- | --- | --- |
| `product-design` — let context explain; stillness and reduced motion | Keep the current metric and period controls; do not add a second “selected day” heading. | Show only the date and useful rate/delta. Pointer feedback only; no keyboard animation. | Check both themes and reduced-motion rendering in Vite. |
| `codebase-patterns/frontend.md` | Keep state local to the exploration and expose chart selection to its host. | No route, API wrapper, query key, or server changes. | Typecheck and inspect the Home exploration. |
| `better-accessibility` | The chart remains keyboard and touch inspectable after tooltip removal. | Preserve the range's accessible name and update `aria-valuetext` with date, active-brand rate/delta, and comparison rate when present. Do not add a live region that announces every pointer movement. | Tab, arrow-key, touch, focus, and dismissal checks. |
| `better-layout` | Keep the metric in its current leading position and ensure the selected date wraps safely. | Reuse existing header layout and spacing; no new card or axis. | Check 320px and 200% zoom. |
| `better-writing` | Use the existing mention-rate language and standard `pp` unit. | Add a visible date; avoid explanatory chart instructions and repeated period labels. Expand “percentage points” in accessible text. | Review exact copy in context. |
| `better-typography` and `better-colors` | Preserve current metric and metadata roles and semantic direction colors. | Use tabular figures and existing positive/negative theme tokens. | Check contrast and wrapping in light/dark themes. |
| `better-ui` | Preserve marker, badge, and controls' established hit areas. | The endpoint is a label, not a new control; no new dependency. | Inspect pill/marker alignment at desktop and mobile widths. |
| `frontend-design` and `ponytail` | Keep the reference's useful relationship without transplanting finance UI. | Reuse `SignalChart`, `HomeSignal`, `BrandPill`, and existing fixtures. | Ensure no unnecessary abstraction or package is introduced. |
| `vercel-react-best-practices` | Keep pointer data local and render only on inspected data-point changes. | Avoid per-pixel animation state and animate a wrapper/marker with transform and opacity. | Confirm chart inspection remains responsive. |
| `emil-design-eng` and `animate` | Motion purpose is selected-state feedback and clearer chart updates. | Target 140–180ms ease-out for marker/headline; up to 220ms for range/filter plot transition; interrupt on a new selection; no keyboard animation; honor `prefers-reduced-motion`. | Play range/filter and point selection in a browser, including rapid changes. |

## View and component specification

```text
Current period:  [45.8%]  [↑ +8.2 pp]
                 Sep 14 – Oct 13
                 [7D] [30D] [90D]

Inspecting:      [54.2%]  [↑ +1.7 pp]
                 Oct 04
                 [existing chart, crosshair, selected halo, end badge]
```

The selected state replaces the period summary in place. Do not add the words “selected day” or repeat the period label. The lower x-axis date range continues to identify the period; while inspecting, the date beside the headline identifies the exact point.

### Components and state

- `HomeSignal` remains the owner of the optional inspected index and its `scopeKey`. Pass `selectedIndex` and `onSelectedIndexChange` into `SignalChart`. The chart continues to own the input source (`pointer`, `touch`, `keyboard`) so it can suppress keyboard motion and retain touch dismissal behavior.
- Derive the headline from `series[selectedIndex]` while the selection scope matches; otherwise use existing `summarize(...)` output. Derive a selected point's signed delta from its `changePoints` value. Keep aggregate `stats.changePoints` for the unselected period state.
- Extend route-private `dailySeries(...)` output with `changePoints`: for each day with a non-null rate, subtract the closest earlier day in the same prompt/engine scope that has completed answers. The helper must include the prior completed day just before the visible window so the first point has a delta when data exists. Return `null` when the selected day has no completed answers or there is no prior completed day.
- Round visible rates and deltas to one decimal, as in the existing metric. Keep semantic color and arrow behavior. Add locale copy for the compact signed `pp` value and the accessible full phrase “percentage points.”
- Remove the visual `role="tooltip"` element and its measurement/placement state from `SignalChart`. Keep the crosshair, selected brand/competitor markers, endpoint label, range input, and `aria-valuetext`. Include selected date and both available series' rates in the range's value text.
- Clear the selected point when `scopeKey` changes. Pointer leave clears pointer inspection; touch selection remains until outside press or Escape; keyboard blur/Escape clears. A day with no completed answers shows `—` and its date, with no delta.

### Motion and geometry

- Pointer selection: transition the selected marker's position/halo and the headline's opacity/transform over 140–180ms with existing `var(--ease-out)`. Update the underlying value immediately; do not count numbers upward. A new pointer selection retargets/cancels the prior animation.
- Date-range, engine-filter, or comparison changes: transition the plot wrapper and endpoint badge together over 220ms, using the existing `var(--ease-out)` token. Reset selected date to the period headline. Keep this effect on the plot wrapper, not the SVG path. No animation for keyboard-triggered changes or under `prefers-reduced-motion: reduce`.
- Set the x-coordinate domain to `0…(plotWidth - 28px)`. Use the same domain when mapping pointer position to a date so the reserved trailing area still selects the last day. Keep the badge connected to the final point and keep it inside the chart bounds.
- No loading/error state changes: this is a deterministic local-fixture exploration. Existing no-data and partial-answer states remain, and a missing day is represented with the existing no-period-answer semantics.

## Data and code contracts

The data path is local and already exists:

`HomeSignal records + VisibilityScope → dailySeries(records, brand, scope, target) → selected daily point → headline and SignalChart`

`dailySeries` currently returns `{ day, date, mentioned, completed, rate }`. Extend each point with `changePoints: number | null`. Apply the same prompt IDs, engine IDs, and completed-answer definition as the existing `inScope` and `rateFor` helpers. Do not change production `VisibilityResult` or OpenAPI contracts. The nearest earlier valid point may skip calendar days with no completed sample answers.

## Ordered implementation tasks

| Order | File and symbol | Concrete edit | Depends on | Observable result |
| --- | --- | --- | --- | --- |
| 1 | `ui/src/routes/dev/explorations/-lib/prompt-visibility.ts`, `dailySeries` | Add the selected day's `changePoints`, using the same filters and the nearest earlier completed day, including one before the visible range. | None | Each visible day has a valid signed delta or `null`. |
| 2 | `ui/src/routes/dev/explorations/-components/home-signal.tsx`, `HomeSignal` | Own selection scoped to `scopeKey`; derive headline rate/date/delta from selected daily data; pass selection state and callback into `SignalChart`; restore aggregate values on clear. | 1 | Scrubbing updates the existing headline without changing the surrounding layout. |
| 3 | `ui/src/routes/dev/explorations/-components/signal-chart.tsx`, `SignalChart` | Accept controlled selected index, preserve pointer/touch/keyboard behavior, remove tooltip UI/measurement/placement, update accessible value text, and reserve 28px after the last data point. | 2 | The chart has a selected halo and crosshair without an overlay; its last point has consistent trailing space. |
| 4 | `ui/src/routes/dev/explorations/-locales/prompt-visibility.en.json`, metric copy; `ui/src/routes/dev/explorations/-home-context.css` | Add concise pp copy and headline/date selected-state styles; implement pointer-only, reduced-motion-aware selection and plot transitions using existing tokens. | 2–3 | The rate and delta change with the marker; controls remain still for keyboard/reduced-motion input. |

## State and failure behavior

| State or trigger | Visible content and exact feedback | Enabled actions | State/focus/data outcome |
| --- | --- | --- | --- |
| Initial or pointer leave | Existing period rate and period-over-period delta; period dates remain below chart. | Existing period, engine, and competitor controls. | No selected index. |
| Pointer scrubbing | Headline shows selected date's daily rate, signed `pp` delta, and compact date; marker and crosshair follow; no tooltip. | Continue scrubbing. | Update only when the pointer crosses to another day; leaving restores aggregate. |
| Touch tap/drag | Same selected-day headline and chart marker. | Continue drag; tap outside or Escape dismisses. | Keep selection latched until existing dismissal behavior. |
| Keyboard range inspection | Same selected-day headline and accessible date/rate/delta; no motion. | Arrow keys change day; Escape or blur clears. | Keep focus on the range input. |
| Selected day has no completed answers | `—` rate and selected date; no pp delta. | Continue to another point. | Do not fabricate zero. |
| No earlier completed day | Selected-day rate and date; no pp delta. | Continue to another point. | Delta is `null`. |
| Period, engine, or competitor changes | New aggregate headline; chart and endpoint badge transition for pointer changes. | Existing controls. | Clear selection; derive new series under the new `scopeKey`. |
| Reduced motion | All values and marker positions update immediately. | All existing inputs. | No transform motion. |
| No tracked/completed data | Existing no-data state. | Existing recovery/clear controls. | No chart inspection state. |

## Acceptance and verification

- On 7D, 30D, and 90D, hover across at least three dates: the marker, visible date, rate, and day-over-day pp delta all refer to the same day; leaving returns the period aggregate.
- Verify the first visible day uses its closest earlier completed data day when available; missing-sample dates are skipped; no prior data yields no delta.
- With a competitor selected, keep both lines visible and include both same-day rates in the range input's accessible value text; the headline remains the active brand's rate and day-over-day delta.
- Confirm the old tooltip no longer appears or obscures the selected marker. Confirm the last data point has 28px trailing space at desktop and 320px width, and the trailing area selects the last date.
- Use touch tap/drag and outside/Escape dismissal; use keyboard arrows and Escape/blur. Keyboard updates are immediate and focus remains on the range input.
- Check light/dark themes, positive/negative/null delta color, 200% zoom, and `prefers-reduced-motion: reduce`.
- Run `pnpm --dir ui check`, `pnpm --dir ui check-types`, and `pnpm --dir ui build`. The repository's discovery instructions assign rendered UI verification to the user; perform the visual and interaction checklist at `http://localhost:5173` after implementation rather than claiming source checks prove it.

## Executor handoff

Implement only after the user reviews this plan. Keep edits to the four files in the ordered tasks, reuse existing fixture helpers and CSS tokens, add no package, and do not change production API behavior. If `dailySeries` has acquired a new caller or return contract, report that specific mismatch before editing the shared helper.
