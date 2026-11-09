# Koi interface system

The production UI uses Hanken Grotesk Variable, semantic Tailwind tokens, Hugeicons and shadcn
Base UI primitives in `ui/src/components/ui`. Product compositions belong in
`components/blocks`; routes use these primitives without overriding their control
size or type scale.

## Current direction and provenance

The historical `home-context` exploration is available in Git at `50a45b7` under
`ui/src/routes/dev/explorations`. It is a historical reference, not a running route.
`plans/ui/home-context-fidelity.md` records its production migration;
`plans/ui/tailwind-and-inter.md` supersedes its Calibre typography. The prompt bulk,
preview and chart plans record later scoped decisions.

The competitor table shows name (with an "Unconfirmed" or "Suggested" chip only when
it applies), domain, 30-day visibility and the gap to the brand, with its action menu,
status filter and server pagination (`plans/ui/ui-review-2026-11.md`). Narrow containers
render the same data as stacked rows instead of scrolling the table. The prompt checkbox now uses shadcn; its 16px visual box and 28px
hit target preserve the approved compact geometry. The specialized visibility chart
and its native range input remain intentional product compositions.

## Shell and composition

Header and main share an outer maximum width of 1376px, with 16px horizontal padding
below `sm` and 28px from `sm`: desktop content is at most 1320px wide. Settings uses
the same geometry. Header actions have 32px visible geometry with 40px hit targets; icons remain compact.
The account control has a border and a transparent idle background in both themes.

Home presents the visibility signal and a "Recommended instead" rival list side by side
from `xl` (signal ≈2/3, rivals 22rem), stacked below that, then prompt management with a
24px gap. With no confirmed prompts the signal shows one sentence and one action. Operational collection blockers remain visible with confirmed prompts and with
existing analytics. Use ordinary text and contextual recovery actions, not another
metric card. Group content with spacing and alignment. Do not add decorative shadows,
boxed metadata or separators between ordinary fields.

Light and dark share layout and hierarchy. Use semantic colors (`background`,
`foreground`, `surface`, `muted-foreground`, `positive`, `negative`, `destructive`).
The PNG Koi mark is unchanged in both themes; primary actions stay neutral.

## Controls and typography

- `Button`: use existing variants and CVA sizes. `circle` is 40×40px; `pill` has a
  minimum height/width of 40px and 8px horizontal padding. `circle-sm` preserves a
  32px visible circle with a 4px hit extension on each side; `icon-outline` supplies
  its border without the filled dark background of regular outline buttons. Ghost buttons support
  `aria-pressed` for filters. Use `render` for menu triggers and navigation links.
- `Checkbox`: 16px visual box with a 6px hit extension on each side, placed inside
  a 28px wrapper. Checked and indeterminate states remain visible in forced colors.
  Accessible name and state are required; Tab/Space operate the control. No animation.
- `RadioGroup`: 16px circle with a hit extension, same focus outline as `Checkbox`. Arrow keys
  move the choice. Setup wraps each item in a bordered label; the border is the control, not
  grouping, and the selected border uses `foreground`.
- `Select`: use for compact status and page-size choices; do not hand-style native
  selects in routes.
- `Input` and `Textarea`: 16px below `sm`. Input is 14px from `sm`; Textarea is 14px
  from `sm` and 12px from `md`. No transform-based size compensation.
- `Table`: use shared table primitives. Pass localized `scrollHint` for potentially
  overflowing data; the hint and keyboard-focusable scroll region appear only when
  content actually overflows. `DataTable` supplies this for product lists.

Use Hanken Grotesk throughout with the `text-body`, `text-section`, `text-title` and `text-metric` roles; numbers use `tabular-nums`. UI copy lives in locales, including
answer status labels. Do not expose raw outcome enum values as labels.

## Collection state

Answer state resolves from the saved result, the run state and the run's requested
engines. Errors precede saved terminal outcomes; excluded engines are `Not requested`;
unresolved active runs are `Pending`; unresolved terminal runs are `Not collected`.
An empty saved answer is never fabricated into a pending response or a brand outcome.

Home and prompt detail share the latest-run query. It polls every 5 seconds during
collection and every 60 seconds at rest after tracking starts, with no background
polling. Detail history and analytics refresh during collection, including one final
refresh when an observed run finishes. URL selection survives these updates.

## First Look wait

Between setup and the first answer, one centered column shows a stable heading and three steps:
read the site, draft competitors and questions, ask the engines. Only saved data advances a step,
never a timer or an estimate, and a finished step keeps what it found. The price column appears with
the first verdict (`plans/ui/first-look-wait.md`).

## Billing without a subscription

With no subscription there is nothing to manage, so billing is one centered panel: what it costs
today, what it costs after, the allowance control, the action. A subscription in any state, including
canceled, keeps the two-column page with its status and management.

## Motion and verification

Stillness is the default. Button pointer press uses the existing 160ms ease-out,
0.97 scale; popup triggers and focus-visible keyboard activation do not scale.
Reduced motion removes transforms. Theme changes suppress transitions.

Validate changed surfaces in both themes at 1440, 960, 390 and 320px, keyboard-only,
200% zoom and reduced motion. Keep full data and actions reachable inside overflowing
tables without horizontal page scroll. Browser tests use simulated APIs; live provider
collection and Safari iOS input zoom require separate environment/device verification.

## Visibility chart contract

Summary is the single chart style for both Home and prompt detail: 220px height with no grid, axes or scale labels (the Family chart: the line, its endpoint and the range control only),
4px solid primary stroke, 1.75px dashed comparison, shared endpoint pills and
140ms pointer-marker feedback. Both use the same responsive insets and scope-change
feedback; detail memoizes series and inspection callback like Home. The domain stays
0–100 without clipping half the stroke at either boundary. Null values remain gaps.
Keyboard and reduced-motion inspection remain immediate.

## Lists, evidence and agent hand-off

Product lists share one row language: no rules between rows, one rounded fill on hover,
focus and selection (`DataTable interactive`, `ul.list-rows`). Re-sorted lists animate
with `useFlip`. Narrow containers switch tables to stacked rows with the action menu always
visible; no list hides actions behind horizontal scroll.

The run dialog is the audit layer: the brand outcome stated in words ("Koi cited"), the
answer with the brand and competitor names underlined in place, competitors with
de-duplicated evidence kinds, and sources by site with the brand's own site first. "Copy
for your agent" writes a markdown brief (`lib/agent-brief.ts`) from the saved answer; no
service is involved, so it works on self-hosted instances. Dialog content uses no inner
panels so light and dark keep the same hierarchy.

## Prompt states

A prompt is in exactly one tab: Tracking, Drafts or Stopped. Each row offers one way out of the
list, chosen from its data rather than by the user: a prompt with saved answers is stopped and
can be resumed with its history intact; a prompt with no answers is deleted, with a confirmation
because that cannot be undone. Stopping and resuming need no confirmation.

## Public landing

`ui/site` is the page at `getkoi.co`: a static build that hydrates, so crawlers read its copy as
HTML and self-hosted binaries never ship it. It uses the app's primitives; prices come from
`pkg/pricing/catalog.json`. These exceptions apply to the landing only:

- The page is true black and dark only (`:root.dark` in `site.css`), which is deliberately not the
  product's colour. The product keeps its own greys inside `.site-product`.
- The hero shows the summary page as a still mock (`ui/site/summary`): the app's own blocks
  (headline, chart, chart controls, rivals, `DataTable`, `EngineMeterCell`) on data recorded from the
  seeded Koi brand (`summary-data.json`). It is inert, so it routes, fetches and saves nothing and
  scrolls past as cheaply as an image. Above 720px it is laid out at 1320px and scaled; below that it
  takes the phone's width, as the app does. Keep its markup in step with the summary page.
- The pond (`ui/site/pond`) appears twice: in pricing, one koi per prompt, and as a fixed shoal behind
  the closing call to action. It is a halftone on black, in staggered rows so hundreds still read as
  fish, beside a panel it never swims under. Orange means AI named you; nothing else on the page is
  orange except the mark. It pauses off screen and holds still under reduced motion.
- Pricing says you pay for prompts, not features. It names no competitor and has no comparison
  table.
- The page is 1360px wide (`max-w-340`). The hero is centred: headline, a lead with the four engine
  marks set inline in place of their names (alt text carries them), then the agent pill.
- Between the hero and pricing sits one bento (`Bento` in `app.tsx`, rules from Stageshow's
  `references/bento.md`): its claim is that the founder keeps the agent they already use and Koi tells
  it what to fix. Rounded tiles on a 16px gap, run-in copy at the top of every tile. The setup
  scene is the only window with chrome; the fix scene is a bare card; the rest are a collection
  (engine marks, for the daily re-check), namesake verdicts, a saved answer and, per question, the
  competitor named instead. Free-check and trial terms live in pricing and the questions, stated
  exactly: the first check needs no card, the trial does. Stages sit
  on the copy's gutter. The setup window bleeds off the tile's right and bottom edges; the fix card
  sits whole inside its tile.
- The hero headline (`.site-hero`) and section titles are medium weight, not semibold. The summary mock
  sits in one frame (`.site-app-frame`) with a single lighter edge, because the app's greys alone
  disappear on true black, and fades out at the bottom where the frame crops it. Headings do not animate in on scroll; only the hero arrives once on first paint.
- The agent call to action is a pill (`.site-agent-pill`): the agents it works with as overlapping
  marks (`AgentStack`), beside the button that copies the prompt.
- Open source is its own short section: one sentence (the Cloud code is on GitHub; self-hosting is
  free, live answers need your own Cloro account), the repository and the self-hosting guide. It
  does not repeat prices or the commands; the guide holds them.
- Engines, agents and competitors appear as their real logos in color (`ui/site/logos.tsx`, sources
  in `assets/logos/SOURCES.md`). The name stays as the accessible label. The app keeps monochrome
  `EngineIcon`.
- `.site-display` and `.site-heading` are larger than the product's type scale.
- The agent setup and fix scenes are Stageshow scenes in `ui/site/demos`, on the kit vendored in
  `ui/site/kit`. Each has a phone design and a poster, and plays once when it comes into view.
- Frequently asked questions are the ones buyers ask (what it is, how answers are collected, engines,
  trial, cancel, self-hosting), not objections we write ourselves; they also ship as `FAQPage` JSON-LD.
  Claims about collection follow Cloro's documentation: real browser, logged out, the brand's country.
- Everything except the pond and the hero chart is readable without JavaScript.

`ui/site/public/SKILL.md` is what the "Start with an agent" prompt points to. Keep its steps in
line with `skills/koi-setup/SKILL.md`.
