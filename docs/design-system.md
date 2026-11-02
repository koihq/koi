# Koi interface system

The production UI uses Inter Variable, semantic Tailwind tokens, Hugeicons and shadcn
Base UI primitives in `ui/src/components/ui`. Product compositions belong in
`components/blocks`; routes use these primitives without overriding their control
size or type scale.

## Current direction and provenance

The historical `home-context` exploration is available in Git at `50a45b7` under
`ui/src/routes/dev/explorations`. It is a historical reference, not a running route.
`plans/ui/home-context-fidelity.md` records its production migration;
`plans/ui/tailwind-and-inter.md` supersedes its Calibre typography. The prompt bulk,
preview and chart plans record later scoped decisions.

The UI consistency correction explicitly retains the four-column competitor table,
its action menu, filters and server pagination. It supersedes the exploration's
competitor list. The competitor table retains a 640px minimum width so brand names
wrap into readable lines rather than single characters; narrow screens scroll within
the table. The prompt checkbox now uses shadcn; its 16px visual box and 28px
hit target preserve the approved compact geometry. The specialized visibility chart
and its native range input remain intentional product compositions.

## Shell and composition

Header and main share an outer maximum width of 1376px, with 16px horizontal padding
below `sm` and 28px from `sm`: desktop content is at most 1320px wide. Settings uses
the same geometry. Header actions have 32px visible geometry with 40px hit targets; icons remain compact.
The account control has a border and a transparent idle background in both themes.

Home presents one full-width visibility signal, then prompt management with a 24px
gap. Operational collection blockers remain visible with confirmed prompts and with
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
- `Select`: use for compact status and page-size choices; do not hand-style native
  selects in routes.
- `Input` and `Textarea`: 16px below `sm`. Input is 14px from `sm`; Textarea is 14px
  from `sm` and 12px from `md`. No transform-based size compensation.
- `Table`: use shared table primitives. Pass localized `scrollHint` for potentially
  overflowing data; the hint and keyboard-focusable scroll region appear only when
  content actually overflows. `DataTable` supplies this for product lists.

Use Inter throughout; numbers use `tabular-nums`. UI copy lives in locales, including
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

## Motion and verification

Stillness is the default. Button pointer press uses the existing 160ms ease-out,
0.97 scale; popup triggers and focus-visible keyboard activation do not scale.
Reduced motion removes transforms. Theme changes suppress transitions.

Validate changed surfaces in both themes at 1440, 960, 390 and 320px, keyboard-only,
200% zoom and reduced motion. Keep full data and actions reachable inside overflowing
tables without horizontal page scroll. Browser tests use simulated APIs; live provider
collection and Safari iOS input zoom require separate environment/device verification.

## Visibility chart contract

Summary is the single chart style for both Home and prompt detail: 220px height,
4px solid primary stroke, 1.75px dashed comparison, shared endpoint pills and
140ms pointer-marker feedback. Both use the same responsive insets and scope-change
feedback; detail memoizes series and inspection callback like Home. The domain stays
0–100 without clipping half the stroke at either boundary. Null values remain gaps.
Keyboard and reduced-motion inspection remain immediate.
