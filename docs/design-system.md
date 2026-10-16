# Koi app design system

The production app uses the home-context exploration as its visual reference. The authenticated workspace shows one active brand. Brand ownership and brand-scoped APIs remain in place.

## Purpose and composition

The founder's first view answers how visible the brand is, which competitors appear stronger, which tracked prompts drive that picture, and where to inspect an actual answer. Home leads with one visibility signal, comparison rows, and then prompt management. Competitors and brand settings are adjacent destinations in the compact header. Prompt detail owns saved engine answers and citations.

Use open canvas space for page structure. A surface is appropriate for the main signal, forms, and collected answers. Avoid a grid of summary cards. The header and main content share a 1320px maximum width, with 16px mobile and 28px desktop side padding.

## Tokens

The source of truth is `ui/src/styles.css`. Light and dark share the same structure and semantic roles. The user's preference is stored as `koi:theme:v1`; dark is the default.

| Role | Light | Dark | Token |
| --- | --- | --- | --- |
| Canvas | `#f4f4f6` | `#121316` | `background` |
| Main text | `#202126` | `#e4e4e8` | `foreground` |
| Raised surface | `#ffffff` | `#0c0c10` | `surface` |
| Quiet text | `#60626b` | `#a5a5af` | `muted-foreground` |
| Hairline edge | `#d5d6dc` | `#29292f` | `border` |
| Positive change | `#35604b` | `#a8c4b7` | `positive` |
| Negative change | `#854753` | `#b79ba2` | `negative` |

The Koi orange remains the mark's accent, never the primary button fill. Controls use the shared shadcn primitives and their semantic tokens. The brand icon uses Logo.dev when configured and initial letters otherwise.

## Type and spacing

The app uses Inter Variable throughout. Headings retain the exploration's compact size and spacing; numeric readings use tabular figures. Avoid monospace for data.

The signal and prompt surfaces follow the exploration's full-width stack with a 24px gap. Use 24–28px padding on desktop and 16px on mobile. The graph has no enclosing grid; its end label and the period controls carry the reading context. Competitor rows sit below the graph in the signal surface.

## States and data

Confirmed live prompts and competitors are the only input to temporary analytical values. The mock generator in `ui/src/lib/mock-visibility.ts` is deterministic across navigation and filters. A brand with no confirmed prompts gets an action-oriented empty state. Review and drafting continue to use the real backend state and optional services.

Saved run answers, outcomes, brand appearances, and citations always come from the visibility API. Prompt detail lists live and deleted prompt snapshots with pagination. Collection errors and pending cells retain their own states; they are not counted as answers.

## Interaction

The graph supports pointer inspection and a keyboard-operable range control. Time range, engines, and competitor comparison update the same scope. Competitor rows select the comparison. Prompt rows expose a short hover or focus preview and link to detail; deletion remains a confirmed action, and archived prompts with saved runs stay reachable.

Motion is restrained: feedback and state indication only. Never animate theme changes or page navigation. Honor reduced motion. Focus remains visible on icon actions, filters, graph inspection, and result links.

## Responsive behavior

The header keeps the brand and account controls visible at narrow widths. Filter controls wrap. Chart content scales within the viewport. Prompt rows become stacked records on mobile so the question and its data remain visible without horizontal scrolling. Prompt detail changes from an engine rail to a horizontal engine list on mobile. The same content and action order is preserved in both themes.
