# Koi MVP exploration: extend the established style

## Baseline and scope

- Date: 2026-10-15. Baseline: `021699837365f08398616163124d550400c3587a`; working tree clean before this document.
- Target: `/dev/explorations/home-context`.
- Status: **Exploration implementation complete; manual visual review pending.**
- This document records the approved restart plan and the implementation result below.
- The user removed the previous implementation because it changed the established visual style. The current exploration is the visual authority. Functional parity comes from the existing production routes.
- Preserve the previously selected exploration-first scope: local sample data, four destinations, matching Settings preview, preview-only logout. Production integration follows separately.
- Current instructions supersede older discovery documents that propose an agent brief, global search, or a different navigation structure.

The founder's tasks are: understand visibility, inspect answers, manage tracked questions, and manage alternatives. Each destination should answer one of those jobs with the smallest useful set of controls.

## What changes in this restart

The implementation must extend the current visual components in place. Matching a palette alone is insufficient. It must preserve the proportions, weight, density, surface treatment, and interaction geometry that make the current exploration recognizable.

1. Clean up the existing shell and Visibility before adding page content.
2. Add Mentions, Prompts, Competitors, and Settings one at a time, using the visual contracts below.
3. Complete their local interactions and map every production feature to its destination.
4. Treat production integration as separate work with explicit API gaps. Do not describe a fixture preview as live feature parity.

Each slice gets a focused diff and a manual visual checklist. Do not rebuild all screens and a simulation framework in one change. This ordering is a verification strategy, not an added permission requirement.

## Research and decisions

Earlier inspected results from the same plan remain useful. This restart also performed these Mobbin searches and inspected the returned images:

| Lane | Actual query | Result and decision |
| --- | --- | --- |
| Project reference / finance analogy | `Fey watchlist stock list with compact rows and floating bottom navigation` | [Watchlist](https://mobbin.com/screens/f4b1fb8e-1db1-407c-a1b1-41ec1eb312c7), [selection state](https://mobbin.com/screens/2075240a-99a1-4224-b0d9-bcee4c29b30f). Compact aligned rows sit on a dark surface; local controls stay subordinate. Borrow list density and alignment. Do not add financial metrics or a bulk-selection toolbar. |
| Same job | `Peec AI prompts page question rows status and add prompt control` | Returned [StackAI prompt library](https://mobbin.com/screens/a249b577-f7e5-4f23-a50c-c9d0fcfeb3e3). It shows a reusable prompt card, not a list of monitored questions; reject this match. See corrected search below. |
| Same job, narrower follow-up | `Peec AI Prompts table with prompt text and visibility columns` (deep search) | [Prompts](https://mobbin.com/screens/8ab182d6-4ef5-4e5d-84a4-a029728d75fa), [populated prompts](https://mobbin.com/screens/7f74462c-ab73-490e-a192-972caf8ae016). Borrow full question text, an explicit add action, and a distinction between active and suggested items. Reject the nested sidebar, many filters, sentiment/position columns, and export toolbar. |
| Another sector / contextual evidence | `Intercom inbox conversation list with selected conversation in right side panel` | [Conversation](https://mobbin.com/screens/b1a806f1-b212-45a8-b525-fea3fa01b961). List context remains visible while reading details. Borrow that relationship only. Reject its multi-rail layout and dense properties panel. Koi already has a suitable glass answer panel. |

Reused research from the original plan:

- [Fey overview](https://mobbin.com/screens/dc5328bd-98d9-46b1-9b18-7ee5805fb878): establishes the current overview's asymmetric composition and material hierarchy.
- [Linear account menu](https://mobbin.com/screens/e8fe60b4-6db4-4485-99b8-3395d69ea7db): account actions in a small anchored menu. Use Koi's existing brand-menu appearance.
- [Fey settings](https://mobbin.com/screens/ffe83560-68ff-49c1-8c6b-d0ad248eed94): a quiet form within the product's visual language.
- Earlier Visitors/Fathom searches did not return convincing relevant matches. No design decisions depend on those results.

These images establish visible relationships, not exact CSS, accessibility, motion, or usability. Measurements below come from the current repository or are explicitly identified as new choices.

## Visual contract: preserve this exactly

Authority: `ui/src/routes/dev/explorations/-home-context.css`, `-components/home-signal.tsx`, `-components/brand-switcher.tsx`, and `docs/fey-visual-system.md` at the baseline commit.

| Element | Contract |
| --- | --- |
| Canvas and shell | Keep `.home-context-study`: `36px 5vw 116px` padding; `.hc-header` and `.hc-main` capped at 1320px. Header bottom margin 40px. |
| Typeface | Keep `--hc-font`: local Calibre, Helvetica Neue, Helvetica, Arial. No font substitution or newly loaded font. |
| Type roles | Keep 12px metadata, 13px compact UI, 14px body, 16px lead, 20px detail title, 36px metric via existing `--hc-type-*` tokens. Weights remain 400/500/700 in their existing roles. |
| Overview | Keep `.hc-evidence` columns `0.88fr / 1.12fr`, 24px gap, chart/ranking/engine column and recap/answer column. No new visible page-title block above it. |
| Content material | Reuse `--hc-surface`, 10px content radius, existing faint inset highlight. Do not introduce generic bordered cards or shadows on each management row. |
| Glass | Reuse `--hc-glass`, `--hc-glass-edge`, and existing menu/panel styles. Glass belongs to floating controls and overlays. |
| Answers | Preserve `.hc-answer-row`, its 24px/28px padding, 14px feed gap, metadata/question/outcome order, brand pills, and current weights. |
| Detail panel | Preserve `.hc-answer-panel`: right/top/bottom 20px; width `min(430px, calc(100vw - 40px))`; padding 28px; gap 24px. Keep the existing modal behavior. Do not substitute a 480px inline inspector. |
| Brands | Keep `BrandSwitcher`, `BrandPill`, `HomeBrandText`, logo fit, fallback initials, overlap, and selection treatment. |
| Dock | Keep glass capsule, 6px padding, 3px gap, 40px desktop targets, current icon size and hover/focus labels. Only destinations and active-state selection change. |
| Responsive | Keep existing 960px and 640px rules. At <=640px: shell `20px 16px 100px`, header gap 24px, overview stacks with 16px gap, surface horizontal padding 20px. |
| Motion | Preserve existing motion. Add no entrance, navigation, chart, or inspector animation. |

New CSS belongs in the same scoped `hc-*` system. Add only selectors needed for the new compositions. Do not create a parallel `mvp-*` theme, replace the stylesheet, change global primitive defaults, or add broad unscoped selectors.

The scoped Fey exception in `docs/fey-visual-system.md` takes precedence over the older Inter-only production guidance for this exploration. It does not authorize changing the production theme.

## Navigation and shell

Replace the top-right `Demo data` text with an icon-only account pill. New geometry: 36px by 28px visible capsule, 16px user icon, existing `--hc-pill` material, inside a 44px non-overlapping hit target. Use an existing Hugeicons user icon, verifying the installed export. Accessible name: `Account menu`.

Use `DropdownMenu`, `DropdownMenuTrigger`, `DropdownMenuContent`, and `DropdownMenuItem` from `@/components/ui/dropdown-menu`. Align the menu to the pill's trailing edge with an 8px offset. Reuse `.hc-brand-menu` material and type; 200px minimum width, 6px inset, 36px desktop menu rows. Exactly two entries: `Settings`, `Log out`. No profile/billing/team section.

Settings opens the matching exploration view. Log out changes only the preview to `Signed out of this preview` with `Return to preview`; no real cookie/session mutation. Real logout remains the existing `api.auth.logout()` flow when this design later reaches production.

Dock order: **Visibility · Mentions · Prompts · Competitors**. Remove gallery, brief, and global-search controls from the product navigation. Preserve the icon-only treatment and hover/focus names. Use real TanStack `Link` destinations with validated search parameters; active styling uses `aria-current="page"` rather than `:first-child`. Settings is reached only from the account menu and leaves all four dock destinations available.

Keep the current route and use `view=visibility|mentions|prompts|competitors|settings`; absent/invalid view resolves to Visibility. View changes reset page-specific filters and selection, preserve selected brand and each brand's chosen period, set document title, scroll to the main content, and focus the page heading (`tabIndex=-1`). Browser Back must restore the prior view. No hash links to sections pretending to be pages.

Remove TanStack devtools UI and its localStorage parking helper from `ui/src/routes/__root.tsx`, and its plugin/optimization entries from `ui/vite.config.ts`. Remove only the three devtools packages from the manifest/lockfile; retain Router and Query. Follow the repository's pnpm-store rule before the package command.

## Page specifications

### Visibility: preserve the existing composition

Keep the chart, periods, ranking, engine comparison, recap, and four latest answer tiles in their current positions. Remove the agent-brief tile and download code. Remove the search dialog, search state, search button, and now-unused props/imports/copy. Leave the space below the feed naturally empty; do not fill it with another widget.

The overview remains the baseline `HomeSignal`, edited in place. Keep its current data calculations and chart geometry in the first slice. Any later correction to metric semantics must be explicit and must not change its visual hierarchy.

Latest-answer clicks open the existing glass panel. A compact `View all mentions` text link follows the feed and opens Mentions for the current brand. The existing sample-data provenance remains in answer details/captions despite removing the header's demo label.

### Mentions: the existing answer feed expanded

Reading order: compact title/control row → collection status when relevant → answer feed → pagination. Title `Mentions` uses the existing 16px/700 lead role; do not add a large heading plus a multi-line subtitle. Controls are local collection selection and a `Filters` menu containing engine/outcome choices. No global search.

Use the full existing `.hc-main` alignment and `.hc-answer-feed`/`.hc-answer-row` vocabulary. Questions wrap fully; engine pill, date, outcome, and mentioned-brand icons remain in the existing order. Do not convert the feed to a dense matrix or add KPI cards.

On select, open the **same** answer panel used by Visibility. Inside it: full question → engine/date/provenance → outcome → answer text → mentioned brands/evidence → source links. Use 20px detail title, 14px body, 12px metadata and 24px section gaps. Sources wrap; only http/https URLs become links. The panel scrolls internally with its close action reachable. Escape/close restores the selected row's focus; if it was removed, restore the page heading.

Retain production functionality: latest/older collections, progress, individual errors, partial results, raw answers, appearances, sources, and pagination. Use `Mentioned`, `Cited`, `Not mentioned`, `Pending`, `Failed`, and `Not collected` as textual statuses. Empty raw text is not fabricated into an answer.

### Prompts: a quiet question list

One `.hc-evidence-main`-style surface, 28px desktop padding and 20px at <=640px. Header inside the surface: `Prompts` at 16px/700 left, `Add prompt` right. Gap to controls 24px. A local search and status choice form one wrapping control row. No metric strip or visible source/ID columns.

Rows use 16px vertical spacing, minimum 52px height, 14px/500 question text, and 12px/400 status. Full question text wraps; trailing actions remain reachable. Use whitespace and alignment, without a box or separator around every row. Clicking the question opens editing; an overflow menu holds deletion. Draft rows expose `Confirm` directly.

Show a review instruction only while review is required: `Review these prompts before tracking starts.` with `Finish review`. Optional `Draft prompts` stays secondary. Once reviewed, remove the instruction; retain status where it matters on unconfirmed rows. Local search/status/pagination preserve existing management capability without adding tags, folders, topics, scores, or bulk-selection chrome.

Prompt form: one visibly labeled `Prompt` field, `Cancel`, `Add prompt` or `Save changes`; use the existing validation. Use a glass Dialog with existing detail typography, maximum 430px width and 28px padding. Delete confirmation: `Delete prompt?`, `Past answers will remain available.`, `Cancel`, `Delete prompt`.

### Competitors: extend the ranking's visual vocabulary

Use the same management surface and header geometry as Prompts. Title `Competitors`; primary action `Add competitor`. Rows: `BrandPill` and domain/GitHub identity, quiet status, trailing actions. Use the existing ranking's alignment and compact type. Do not add a logo-card grid, another chart, or rates that the API does not provide.

Keep tracked/confirmed, drafted/unconfirmed, and observed suggestions distinguishable in text. Local search and status filter match Prompts. A short `Suggested from answers` group can appear after tracked rows when viewing all items; it is part of the same surface, not a new card. Preserve server pagination order within each group and do not imply groups contain off-page items.

Suggested rows offer `Confirm` when identity is sufficient, otherwise `Add identity`; `Dismiss` is available. Finishing competitor review never silently confirms observed suggestions. A founder may finish review with no competitors.

Form: name and domain first; `More details` discloses GitHub repository, aliases, and extra URLs. Optional competitor lookup uses a bounded combobox; manual input always works. Reuse production validation and duplicate handling. Delete confirmation distinguishes deleting a tracked competitor from dismissing a suggestion.

### Settings: the same product, a focused form

Place a 640px maximum-width form at the main content's leading edge, with a compact `Settings` title and 24px gap to fields. Use existing `Field`/`FieldGroup`/`Input`/`Combobox` primitives within the exploration tokens. Field order follows current `BrandForm`: brand identity, description, country, optional additional identity fields. One `Save changes` action and inline `Changes saved` feedback.

No account/security/billing features are implied. This preview edits the selected brand's settings locally and keeps them when moving between views. Reuse production validation, including domain-or-GitHub identity requirements. Unsaved field values remain while staying in the form; leaving the view discards them, while saved values persist for this preview session.

## State, interaction, and failure contract

| Situation | Behavior |
| --- | --- |
| Local preview | Existing six brands; deterministic samples; edits survive navigation/brand switches and reset on full reload. No localStorage persistence or network mutation. |
| Loading | Keep shell and page geometry; plain `Loading…` inside the relevant content area. No animated skeleton redesign. |
| Empty prompts/competitors | `No prompts yet` / `No competitors yet` with one add action. |
| Filtered empty | `No matches` with `Clear filters`; retain search and filters. |
| Empty mentions | `No answers yet`; link to the outstanding prompt/competitor review step, or show collection status when ready. Never show an invented 0% result. |
| Error | Inline `Unable to load answers. Try again.` and `Retry`; equivalent resource-specific wording for management. |
| Mutation pending/error | Disable repeated submission, preserve label and entered values, show field errors or `Unable to save. Try again.`; successful save closes the dialog and restores focus. |
| Review | Confirm eligible draft items; only both completed reviews allow collection. Name-only observed competitors require identity before individual confirmation. |
| Drafting | Preserve initial/prompts/competitors modes and queued/running/succeeded/failed/empty outcomes. Retry the failed mode. Optional service absent: manual actions remain available. |
| Historical evidence | Editing/deleting a prompt or competitor never rewrites an old answer, prompt text, or appearance snapshot. |
| Brand/view change | Dismiss panels, reset page-specific filters/page, and prevent prior-brand results from appearing under the new brand. |
| Mobile/zoom | Controls wrap; rows grow; body/questions remain readable at 320px and 200% zoom. Form input text 16px on narrow screens; preserve the compact desktop type. |
| Keyboard/touch | Existing primitive menu/dialog keyboard handling, 2px token focus rings, explicit names. At coarse pointer increase new action and dock hit targets to 44px without enlarging icon artwork. |

Loading/error/drafting edge states should be explicit fixture examples for inspection, not a new simulated backend with timers, polling, random failures, and collection scheduling. The visual slice may use fixed state fixtures; complete local CRUD with direct state updates when implementing interactions. Do not expose developer scenario controls in product chrome.

## Data contracts and production-parity map

The exploration uses fixtures; production hooks define behavior. Use public API types for new records. Preserve the existing `HomeAnswer`/`homeEvidence` data for the first Visibility slice. Add an explicit adapter only where detailed result fields are required, preserving existing brand names, ordering, rates, and dates for baseline comparisons.

```ts
type ExplorationView = "visibility" | "mentions" | "prompts" | "competitors" | "settings";

// Route owns selected view/brand and composes the screen.
// Narrow components receive records and callbacks, not a replacement API client.
type PromptListProps = {
  items: Prompt[];
  onEdit: (prompt: Prompt) => void;
  onConfirm: (id: string) => void;
  onDelete: (prompt: Prompt) => void;
};

// Verified production wrappers for later integration:
api.prompts.list(brandID, { page: 1, pageSize: 20, q: "", confirmed: false });
api.competitors.list(brandID, { page: 1, pageSize: 20 });
api.runs.list(brandID, 1, 10);
api.runs.listResults(brandID, runID, 1, 40);
```

Lists use `ApiListResponse<T>` / generated `Page` types, including empty envelope metadata. Prompt/competitor page sizes are 10/20/50; answers 40. Filter before local fixture pagination; reset page on a filter change. For production, preserve server pagination rather than presenting filtered current-page results as the full collection.

| Capability | Existing source to preserve | New destination |
| --- | --- | --- |
| Prompt CRUD, confirm, filters, pagination | `_authenticated/_app/prompts/index.tsx`, its list/mutation hooks and form schema; `api/prompts.api.ts` | Prompts |
| Competitor CRUD, search, identity, observed suggestions | `_authenticated/_app/competitors/index.tsx`, dialog/list/mutation hooks; `api/competitors.api.ts` | Competitors |
| Review/readiness | `components/blocks/tracking-review.tsx`, `hooks/use-tracking.ts` | Conditional local status in Prompts, Competitors, Visibility/Mentions |
| Optional drafting | `components/blocks/generation-controls.tsx`, `hooks/use-generation.ts`, `api/generations.api.ts` | Secondary actions on management pages |
| Run history/results/status | `_authenticated/_app/mentions/-hooks/use-visibility.ts`, `api/runs.api.ts` | Mentions |
| Raw answer/appearances/sources | `VisibilityResult` from `api/runs.api.ts`, existing visibility matrix/detail behavior | Shared answer panel |
| Brand settings/countries/validation | `components/blocks/brand-form.tsx`, `api/brands.api.ts` | Settings |
| Logout | `components/blocks/workspace-menu.tsx`, `api/auth.api.ts` | Account menu; preview behavior stays local |

When live integration begins, reuse `ApiProvider`, existing TanStack Query hooks and `promptKey`, `competitorKey`, `runKey`, `runResultsKey`, `trackingKey`, `generationKey`. Do not import route-private hooks into another feature: move truly shared logic to `hooks/` when it gains a second live consumer.

Known production gaps: no verified aggregate period-summary endpoint for the overview; result lists do not accept engine/outcome server filters; current production shell does not implement the exploration's six-brand selection. These require a separate concrete integration plan. Do not infer new endpoints or compute global rates from one response page. Live collection requires the instance owner's Cloro configuration; optional OpenRouter/Logo.dev absence cannot block manual management or initials.

## Skill preflight

| Skill/rule | Decision and adjustment | Verification still needed |
| --- | --- | --- |
| Product design / primitives | Scoped established Fey style wins; preserve brand PNG/pills, no widget grid or parallel theme. | Rendered comparison to baseline. |
| Codebase patterns | Route owns composition; small private components, public resource types; no new backend in this pass. | Types, import boundaries. |
| Better layout | One task per destination; contextual overlays; existing breakpoints and aligned surfaces. | 320px, 390px, desktop, 200% zoom. |
| Better typography | Exact existing font stack, roles, weights; 16px mobile inputs. | Actual fallback font, wrapping, text clipping. |
| Better colors | Reuse tokens and existing outcome colors; statuses have text. No silent palette revision. | Rendered contrast including translucent overlays. |
| Better UI | Existing radii, glass, pills, dock; existing motion overrides generic skill recipes. | Hover, selection, touch, overlay geometry. |
| Better accessibility | Native links/buttons; primitive focus handling; meaningful names and error announcements. | Keyboard, screen reader, focus restoration. |
| Better writing | Compact titles, verb-first actions, specific empty/error recovery. | Labels match implemented behavior. |
| Frontend design | Current established composition is the brief; no novel art direction. | Side-by-side fidelity. |
| Ponytail | Reuse existing pieces, direct local state; no generic preview job engine, dependency, or simulation framework. | Diff size and unnecessary abstractions. |
| React best practices | Derived filtering during render, immutable updates, narrow subscriptions; SPA conventions. | State isolation, repeated actions, render behavior. |
| TanStack Router | Validated optional view search, real Links, browser history; preserve deep link without requiring a long default search object at every caller. | Back/forward, reload, invalid values. |

No rendered/interface approval is claimed by this preflight. Primitive motion is not being changed; its specialized implementation skills are not needed for this plan.

## Ordered implementation handoff

All paths below are relative to `ui/src/routes/dev/explorations/` unless otherwise stated.

| Order | Files/symbols | Edit and visible outcome |
| --- | --- | --- |
| 1 | `home-context.tsx`; `-components/home-signal.tsx`; existing two locale files; `-home-context.css` | Remove search and agent brief in place. Existing overview stays visually identical outside the removed elements. Do not rewrite `HomeSignal`. |
| 2 | `ui/src/routes/__root.tsx`, `ui/vite.config.ts`, `ui/package.json`, `ui/pnpm-lock.yaml` | Remove devtools component/helper/plugin/packages; retain routing/data libraries. |
| 3 | `home-context.tsx`; new `-components/account-menu.tsx`; existing stylesheet | Add user pill and two-item menu. Introduce validated optional `view`/`brand` navigation. Dock contains only the four real destinations; no dead links in the completed slice. |
| 4 | `home-context.tsx`; new `-components/answer-feed.tsx`, `-components/answer-panel.tsx` | Extract existing answer markup with unchanged classes/styles. Compose Mentions in the route, reuse panel for overview. Keep exact 430px geometry. |
| 5 | `home-context.tsx`; new `-components/prompt-list.tsx`, `-components/prompt-editor.tsx` | Compose Prompts using the specified one-surface layout and exact current type roles. Add local CRUD/review/filter behavior, reusing validation. |
| 6 | `home-context.tsx`; new `-components/competitor-list.tsx`, `-components/competitor-editor.tsx` | Compose Competitors using the same surface and brand pills; implement identity-aware suggestions and manual forms. |
| 7 | `home-context.tsx`; new `-components/settings-form.tsx` | Compose compact Settings and preview logout/return. Preserve actual brand form fields and validation. |
| 8 | New `-lib/exploration-fixtures.ts`, `-hooks/use-exploration-state.ts`, and `-locales/mvp.en.json` only as needed by steps 4–7 | Small deterministic typed fixtures/direct state; preserve historical evidence. Complete feature/state coverage without changing the selected layouts. |
| 9 | `docs/fey-visual-system.md`; focused tests beside any new nontrivial helpers | Update obsolete search/brief/manual-check descriptions and document precise parity limits. |

Shared source-URL validation can be extracted from `mentions/-lib/visibility.ts` to a shared utility if needed; move its existing tests/imports together. Never hand-edit generated clients or the route tree. No new dependency is required.

Step 3 navigation is completed alongside destination slices; do not ship blank or mislabeled destinations as finished. Provide a reviewable shell/Visibility diff before the wider page changes, without requiring a new visual design.

## Acceptance and verification

Before implementation, capture the baseline for comparison using the same viewport, brand, period, browser, and font environment. Browser testing is assigned to the user by the existing discovery instructions; supply this checklist rather than claiming it ran.

1. At desktop width, Visibility differs only in the requested removals, account pill, dock destinations, and the small mentions link. Chart size, columns, recap weight, answer tiles, brand logos, header position, and panel remain unchanged.
2. Switching views preserves font, canvas, materials, control density, and dock geometry. New pages feel like extensions of this screen. A new generic page-header system or default production-table styling fails this check.
3. Account menu opens with pointer and keyboard, closes on Escape, and restores focus. Settings works in the exploration; preview logout leaves the real session untouched.
4. Dock links support keyboard and Cmd/Ctrl-click. Browser Back restores the prior view. Selected brand and per-brand period survive navigation.
5. Open the second answer on both Visibility and Mentions; the same panel appears. Close it and return focus to that answer. Source URLs and long questions remain readable at 320px and 200% zoom.
6. Add/edit/delete a prompt; confirm a draft; finish review. Historical answer text stays intact. Repeat across two brands and verify isolation.
7. Add a competitor manually with optional services absent; resolve an observed suggestion; dismiss another. Finishing review leaves unresolved observed names unconfirmed.
8. Inspect empty, loading, failed, partial, and drafting fixture states without changing layout. No false 0% for missing data and no fabricated source/answer content.
9. Check 320px/390px, keyboard, coarse pointer, reduced motion, and final-page dock overlap. Inspect actual overlay contrast and font rendering; these remain unverified until performed.

Mechanical checks during implementation: `pnpm --dir ui check`, `pnpm --dir ui check-types`, `pnpm --dir ui test`, and `make ui-build`. Apply `make format` per repository policy. Add focused tests for navigation parsing, filtering/pagination, immutable historical evidence, and confirmation rules only when implementing that logic. Do not add tests asserting arbitrary CSS values as a substitute for visual comparison.

Completed in this restart: clean-baseline verification, source/feature inventory, Mobbin image inspection, skill preflight, revised design handoff. No application build or runtime verification was needed for this documentation-only change.

## Implementation result

Implemented after approval: shell cleanup, account menu, four working navigation
destinations, shared answer feed/panel, local management forms, and matching Settings.
The existing overview was edited in place. A source comparison confirmed unchanged
declarations in 18 retained visual blocks, including columns, chart, recap, answer
rows, dock material, and panel geometry. This is source evidence, not rendered approval.

Validation passed: `make format`, `pnpm --dir ui check`, `pnpm --dir ui check-types`,
`pnpm --dir ui test` (15 files, 54 tests), `make ui-build`, and `git diff --check`.
No browser tests were performed, following the existing discovery instructions.
The [manual walkthrough](../../ui/src/routes/dev/explorations/-docs/mvp.md) lists
the interaction checks, fixture URLs, and production integration boundaries.

Implementation choices: the existing BrandForm accepts optional local country options
without changing its production default; source-URL validation and prompt validation
were moved to shared utilities with compatibility re-exports. Fixed empty/review states
end at collection readiness instead of synthesizing a new run. Draft suggestions are
deterministic and local. No real session, collection, or drafting mutation is made.
