# Fey visual reference for Koi

Status: chosen visual direction for discovery. These are working Koi tokens, not a recovered
copy of Fey's proprietary design system. The user has authorized departing from Koi's
existing visual conventions for this exploration.

## Sources and confidence

Inspect the original product screens, rather than the current marketing site:

- [Fey Home](https://mobbin.com/screens/35b2c386-c5d3-42a7-bcf3-d17798f8c34c):
  quiet greeting, inset dark content surfaces, compact metadata, and a floating dock.
- [Fey Home with watchlist](https://mobbin.com/screens/ccac3be7-3fb5-4ed3-baf6-20229f5ad97a):
  a darker contextual surface, tightly aligned rows, and restrained selected controls.
- [Fey portfolio](https://mobbin.com/screens/439de82b-140c-4673-b36c-1e8f61b015f1):
  comparison and watchlist share a surface with small local controls.

These images establish visual relationships, not exact CSS measurements. Values below
are explicit reproduction choices that the user will evaluate.

## Fonts

Koi currently loads **Inter Variable** from `@fontsource-variable/inter`, and defines it
as `--font-sans` in `ui/src/styles.css`. The Home study now has its own font stack.

[Pixloop's independent reference](https://www.pixloop.ai/design-md/feyapp) identifies
Fey's original typeface as **Calibre**. This is a secondary-source identification, not
verification of the font used in every original application screen.
[Fey's current site](https://fey.com/) is an acquisition announcement and says sign-ups
are closed; it cannot establish the original app's typography.

[Calibre](https://klim.co.nz/fonts/calibre/) is a Klim Type Foundry family with regular,
medium, semibold, and other weights, web font files, and tabular numerals. No Calibre
files have been added. The current Home study tries locally installed Calibre Regular,
Medium, and Bold, then Helvetica Neue, Helvetica, and Arial. On a Mac without Calibre this normally
renders Helvetica Neue. This is an approximation; exact parity needs licensed Calibre webfont
assets and verification against the original app. Other Koi routes still use Inter.

Weight correction, October 14: no Calibre files were found in the repository or the checked
local font directories. Inspection of `/System/Library/Fonts/HelveticaNeue.ttc` found Regular
400, Medium 500, and Bold 700, with no Semibold 600. Requesting 600 therefore selects 700
under the [CSS weight-matching rules](https://www.w3.org/TR/css-fonts-4/#font-style-matching).
The founder clarified that Koi looks too thin/light. The revised treatment uses actual
Medium 500 for content and numbers, and Bold 700 for headings and the recap; metadata
stays Regular 400. The recap's explicit 700 does not itself increase weight on this Mac
fallback, since the previous 600 already selected Bold. The substantial change is moving
body/row text and values from Regular to Medium, and headings from Medium to Bold.
These are deliberate fallback weights, not evidence of Fey's exact CSS. Actual browser
font selection remains unverified. Do not label this implementation an exact Fey font match.

## Surface system

| Role | Working value | Use |
| --- | --- | --- |
| Canvas | `#121316` | Workspace |
| Content | `#0c0c10` with a very faint diagonal highlight | Chart and answer tiles |
| Recap | `#08090b` with a low-opacity warm highlight | Narrative recap |
| Glass | `#252529c9` plus a translucent white highlight | Dock, answer panel, account and brand menus |
| Primary text | `#e4e4e8` | Readable content |
| Secondary text | `#a4a4ad` | Metadata |
| Comparison | Desaturated lavender | Alternative chart line |
| Outcome | Muted sage / rose | Mention outcomes |

Glass uses 24–32px backdrop blur, a subtle inset top highlight, and a soft shadow on
floating surfaces. Content uses deeper surfaces with faint edge illumination. These
are reproduction choices from the screenshots, not recovered Fey CSS values.

## Typography and spacing

This section records the earlier overview treatment. The current single-view role mapping
is in [Unified visibility typography](#unified-visibility-typography-october-16).

Rechecked the [Fey overview](https://mobbin.com/screens/dc5328bd-98d9-46b1-9b18-7ee5805fb878)
at full image resolution on October 13, 2026, alongside
[Fey Home](https://mobbin.com/screens/35b2c386-c5d3-42a7-bcf3-d17798f8c34c) and
[Home with a watchlist](https://mobbin.com/screens/5c326535-4eb7-4652-855f-1c6fe9ecdb91).
The recap paragraph has consistently stronger weight than the news feed. Local
section labels, stock labels, dates, and chart controls are compact. The chart's short
verdict is more prominent than these labels, without becoming a large page heading.

Screenshots do not expose font weights, CSS sizes, device scale, or line-height values.
The following are Koi reproduction choices based on those relationships, not measured
Fey tokens. Sizes use the `--hc-type-*` roles in `-home-context.css`, expressed in rem;
pixel equivalents below assume a 16px root. Line heights are unitless.

| Role | Size | Weight | Line height | Use |
| --- | --- | --- | --- | --- |
| Metadata | 12px | 400 | 1.5 (18px) | Dates, outcomes, captions, chart legend, tooltips |
| Local label | 12px | 500; 700 for headings | 1.5 (18px) | Section headings, engine labels, period controls |
| Compact UI | 13px | 500; 400 for supporting copy | 1.5 (19.5px) | Ranking/engine rows, supporting copy, local actions |
| Body | 14px | 500 | 1.5 (21px) | Answer feed/detail, management rows, brand menu |
| Lead | 16px | 700 | 1.375 (22px) | Chart verdict and view titles |
| Recap | 16px | 700 | 1.5 (24px) | Entire recap paragraph, including its semantic strong text |
| Panel title | 20px | 700 | 1.4 (28px) | Wrapping answer question |
| Metric | 36px | 500 | 1.1 (39.6px) | Visibility rate; percent sign at two-thirds size |

The 11px avatar initials and overflow count remain an exception for the existing 24–26px
circles. Form input text stays 16px at narrow widths to avoid iOS focus zoom. Body tracking is neutral;
only the short lead, panel title, and metric have restrained negative tracking. Changing
metrics retain tabular figures. Headings balance their wrapping; recap and feed copy use
pretty wrapping. Form and answer portals share the same body baseline as Home.

This corrects the previous flat 15px section-heading/answer hierarchy, the mostly regular
recap, and the oversized 42px medium metric. Section labels now sit below the content
they introduce, and the recap gains emphasis through weight rather than extra size.

Layout is capped at 1320px with a 24px column gap and
24–28px surface padding. Content corners are 10px; floating panels are 12px; the dock
is a capsule. At narrow widths the columns stack. No new motion is introduced.

## Additional Mobbin references examined

- [Portfolio](https://mobbin.com/screens/ba4a167d-2b95-49dd-b8d9-fd8ddbeb90dd):
  inspected at full resolution for type proportions, control highlights and dock material.
- [Search overlay](https://mobbin.com/screens/05f87eae-99d3-4d7a-a7c4-c443a99afb0b):
  glass search surface floating over the dashboard.
- [Context panel](https://mobbin.com/screens/8487cb92-3969-4f21-a847-26177466cf9a):
  a right-hand inspection surface keeps the overview in context.

Koi adapts these to brand comparison, engine coverage, and answer inspection. The search
overlay reference is retained as research history; the global search control and agent brief
were removed for the MVP on October 15.

## Home revision: Fey overview composition

Open `/dev/explorations/home-context` on the Vite server. The workspace banner,
large goal editor, tracking-scope form, exploration bar, and session footer were removed
at the user's request. This replaces the earlier header-first study.

The direct composition reference is [Fey's overview](https://mobbin.com/screens/dc5328bd-98d9-46b1-9b18-7ee5805fb878):
- The existing compact brand-avatar switcher and an account pill with Settings and Log out.
- A narrower left surface holding the chart, inline period controls, and compact ranking.
- A wider right column of separate dark recap and answer tiles.
- The same floating glass dock now navigates to Visibility, Mentions, Prompts, and
  Competitors. Settings is reached from the account menu. No gallery, search, or brief
  destination is included in this MVP navigation.

Chart lines compare the selected brand and its first tracked alternative on the same
0–100% scale. Ranking and recap use the same filtered synthetic answer records.
Brand switching restores each brand's selected period. Buyer-question scope defaults
to alternatives for Deja and discovery for Koi; all engines are included.
The overview is now named Visibility. The agent brief and its export have been removed.

No real collection or agent connection is represented. Evidence and answer details are
labeled as demo data; the fixture period ends October 13, 2026. The font stack and its
remaining fidelity limitation are documented above.

Manual checks for the user:

1. Switch brands through the avatars. Check names, comparison, and answers change together.
2. Select 7D, 30D, and 90D. Check the timeline, sample count, ranking, and recap update.
3. Open an answer, then close with Escape. Check focus returns to the selected tile.
4. Open the account pill, visit Settings, and return through the dock.
5. Use the dock to visit all four views; check the selected destination and browser Back.
6. Check 390px and 320px widths for overflow and dock overlap with the last action.
7. Open Mentions, filter by engine or outcome, open an answer, and close with Escape.
8. Compare the desktop layout against the linked Fey overview, particularly column
   proportions, compact type, chart line weight, tile spacing, and dark surface contrast.

No browser testing is performed, per the user's instruction.

Typography checks for the user: compare recap weight with the medium-weight answer feed; check
the chart verdict against small section labels; open the account menu, brand overflow menu,
and an answer to check their type consistency. At 390px, 320px, and 200% zoom, check long
competitor names and wrapping answer titles for clipping. Rendered font selection and
wrapping are not verified by source checks.


## Brand avatar switcher

The greeting and standalone Koi workspace mark have been removed. Brand logos occupy 26px
buttons with 24px logo slots. Initials are the fallback. The collapsed centers are 20px apart,
creating a small overlap with a 1px canvas-colored rim. The image carries its own brand colors;
only missing-logo initials use a neutral muted surface. Logos remain at full opacity;
the button's thin ring communicates selection without dimming brand artwork.

Hovering the group spreads its centers to 32px apart using interruptible 200ms
transform transitions. The group reserves its expanded width so neither neighboring
content nor the hover boundary shifts. Leaving it restores the stack. Keyboard focus
and an open overflow menu keep it expanded; keyboard and reduced-motion changes are
immediate. Touch/coarse-pointer devices show the separated row. Names appear in
tooltips on hover and focus.

The configurable `MAX_VISIBLE_BRANDS` in `home-context.tsx` is 3. Additional brands go
into a +N menu. Selecting an overflow brand places it in the final visible slot;
the displaced brand becomes available in the menu. Existing slots retain their order.
The demo includes Deja, Koi, Mintlify, beehiiv, Tally, and PostHog. The last four replace
the fictional workspace examples at the user's request and have separate question sets,
real alternatives, and goals. Their presence is demonstration data, not a claim of ownership.
Three avatars and a +3 menu are visible immediately. Select an overflow brand and
check that its own chart, answers, prompts, and competitors are shown.
Period selections remain independent for all six brands.

## Brand logos and pills

The [Fey feed on Mobbin](https://mobbin.com/screens/af4a7c92-c022-450b-917e-4b071d4cd920)
and the [overview](https://mobbin.com/screens/dc5328bd-98d9-46b1-9b18-7ee5805fb878)
inform the rounded-logo treatment. The image itself is rounded without an extra dark
image frame. When a name is present, the logo and label share a subtle capsule; icon-only
variants have no capsule. Koi uses the shared
`BrandPill` block for every visible brand reference in this Home study: chart title and
legend, brand/engine comparison, recap, feed metadata and question text,
answer detail, brand menu, and brand tooltips. The stacked selector uses the same
block's icon-only variant to preserve its overlap and interaction geometry.

Content logo slots are 16px. The stacked brand switcher retains its 24px slots and existing
hit areas. Assets with their own background or circular badge fill the slot and are rounded
directly. Transparent symbols occupy 75% of the slot, centered with `object-contain`, without
clipping the symbol or painting an extra background. Koi uses its original PNG with this
inset. A 12px medium label sits alongside with a 4px gap.
Labeled variants have a full-round `--hc-pill` surface (`#1d1e23`), faint inset edge, 2px
vertical padding, 4px leading padding, and 8px trailing padding. Icon-only variants have no
capsule surface or edge. The shared `BrandIcon` owns the same fit rules in its other consumers. Initials render
only when an image is unavailable, so they do not show through transparent logo pixels.
Selection/focus indicators stay on the switcher button. The brand label is a presentational
span inside the existing button or row, not a nested control. Names can wrap on narrow
screens; metadata rows can wrap instead of forcing a logo and date to overflow.

Use `BrandPill` for structured brand fields and `HomeBrandText` for prose containing known
brands or engines; do not print those names as plain text in the rendered Home UI.
`HomeBrandText` preserves the original wording and matches whole names case-insensitively,
without mistaking “toolkit” for Kit. Mention outcomes use icon-only pills with accessible
names and native hover titles. Chart legends retain their line-color markers. Markdown
exports and accessibility descriptions retain plain text rather than image markup.

`BrandIcon` supplies the actual image and keeps the existing initial-letter error fallback.
Logo.dev uses the configured `VITE_LOGODEV_PUBLISHABLE_KEY`, `format=webp`,
and `fallback=404`. It explicitly requests `theme=light` or `theme=dark` according to the
document root's `.dark` class, including theme changes while mounted. Logo.dev's `auto`
is equivalent to `light`; it does not follow the app's theme. No CSS color filters are used.
The central `BrandIcon` overrides distinguish known filled assets from transparent symbols
and map ReadMe's blank `readme.com` image to its valid `readme.io` logo. An optional
`appearance` prop can override fit, backing color, or asset theme when a verified asset needs
special handling. Unknown logos default to an inset symbol; missing images or credentials
retain local initials.
Its [documented CDN parameters](https://www.logo.dev/docs/api-reference/logo-cdn/get-a-logo)
do not include a pill, radius, or label option; image rounding belongs to Koi's CSS.
No new logo service or credential is introduced.

All 18 configured brand/engine domains returned HTTP 200 WebP images when checked on
October 14, 2026. The ChatGPT, Gemini, and Mintlify images were inspected: Logo.dev currently
returns Google's G for the Gemini domain, so its text label distinguishes the engine.
Koi uses its existing unmodified PNG mark. Deja and zsh-autosuggestions have no documented
brand domain or logo asset here, so they retain initials rather than showing GitHub's logo
as if it belonged to those projects. UI rendering remains for the user's manual check.

October 15 contrast check: inspected the configured Home domains as light/dark image pairs.
Tally, ChatGPT, Perplexity, and Typeform provide visible dark variants. Filled logos retain
their own backing colors. ReadMe's original asset returned HTTP 200 with no visible pixels;
the `readme.io` replacement was downloaded and inspected. A separate image contact sheet
checks the 16px content and 24px switcher treatments on light/dark surfaces; browser layout
and interactions remain for the user's manual check.

Mobbin references examined:
- [Confluence avatar stack and account selection](https://mobbin.com/screens/ed0de1ce-bf14-4cfc-a512-d95ac91bc369).
- [Figma compact avatar and dropdown](https://mobbin.com/screens/788651ef-5ef6-4a17-9305-793735da7a5a).
These inform the compact pattern; neither is claimed to implement Koi's exact behavior.

Motion uses CSS transitions with the existing `--ease-out`: 160ms avatar opacity/ring
and scale(0.97) press feedback, 125ms tooltips, and 200ms overflow menu. Menus and
tooltips use their trigger-derived transform origin. Neighboring tooltips skip delay
and animation after the first. Keyboard interactions and reduced-motion settings are
immediate. No dashboard entrance animation is added.

Manual checks: rapidly alternate brands, hover across avatars, Tab and activate with
Enter/Space, close the menu with Escape, and repeat with reduced motion. Confirm active
brand visibility after overflow selection and that brand-specific periods remain isolated.


The overlapping geometry is informed by the [X community header on Mobbin](https://mobbin.com/screens/536ec837-46f2-4217-b7a2-fdc2f30269bc),
which uses partially occluded circles with separating rims. Koi adds active-brand
layering, dimmed inactive states, and a brand overflow menu to that visual pattern.
Selection fades and press feedback retain their prior timings. Expansion moves only
avatar and overflow transforms, preserving the page layout and reduced-motion behavior.


Stack correction: every slot, including +N, uses the same collapsed displacement.
Explicit descending layers place each circle above its right neighbor, independent
of selected state. The active ring and opacity identify selection without changing
the stack order. The overflow circle has an opaque surface and the same 1px rim.


## MVP expansion, October 15

The implementation extends the existing exploration's visual system. The overview's
columns, chart, recap, answer rows, fonts, and 430px glass answer panel retain their
baseline declarations. New management lists reuse the dark content surface, compact
16px headings, existing pills, and 14px body role. Forms and menus use the same scoped
material; production component defaults are unchanged.

See the [implementation plan](../plans/ui/mvp-exploration-restart.md) and
[manual walkthrough](../ui/src/routes/dev/explorations/-docs/mvp.md).
The preview uses local session state and fixed historical samples. It makes no collection,
drafting, or session/logout request. Brand settings use the existing validated form with
local country options. Source links are supported, but baseline synthetic answers have no
collected sources. A separate live integration is needed for period aggregation, complete
server-side answer filtering, and production multi-brand selection.

## Unified visibility typography, October 16

The unified `/dev/explorations/home-context?sort=original&page=1` view keeps the existing
Calibre-local/Helvetica Neue fallback stack. The reading order is mention rate, competitive
insight, prompt evidence, then supporting metadata. Prompt detail, editors, confirmation
dialogs, menus, competitor management, and Settings use the same roles.

Mobbin research inspected [Fey portfolio](https://mobbin.com/screens/f4b1fb8e-1db1-407c-a1b1-41ec1eb312c7)
for compact labels and aligned comparison rows, and
[Fey home](https://mobbin.com/screens/64282196-4e6c-4590-a0d1-07bde4e4819b) for the stronger
narrative insight. An analytics overview query for Visitors returned
[Peec AI](https://mobbin.com/screens/0711fee8-0de2-4f0e-8b7c-16387d22f091) and
[AirOps](https://mobbin.com/screens/27ca7fbd-d9e9-4c1e-9f3f-0ce500c1cef1), rather than Visitors.
Peec AI's quiet column labels inform the evidence table; its dashboard composition is
not adopted. These screenshots establish relationships, not measured CSS values.

The existing `--hc-type-*` tokens now have these roles. Pixel equivalents assume a 16px
root font; sizes remain in rem and line heights are unitless.

| Role | Size | Weight | Line height | Use |
| --- | --- | --- | --- | --- |
| Primary metric | 36px | 500 | 1.1 | Mention rate |
| Detail/dialog title | 20px | 700 | 1.4 | Prompt question, editors, confirmation dialogs |
| Insight/view title | 16px | 700 | 1.4 | Competitive insight, Settings and Competitors titles |
| Main content | 14px | 500 | 1.5 | Prompt sentences, competitor names, editable text, response dates, period-change value |
| Section heading | 13px | 500 | 1.5 | Mentioned in answers, Prompts, Responses |
| Compact data/control | 13px | 500 | 1.5 | Comparison rows, table values, buttons, periods, selects, menu options |
| Supporting prose | 13px | 400 | 1.5 | Explanations, form help, validation and recovery messages |
| Metadata | 12px | 400 | 1.5 | Counts, statuses, chart labels, column labels, provenance dates |
| Local heading | 12px | 500 | 1.5 | Compared brands, appearances, sources |
| Answer prose | 14px | 400 | 1.6 | Answer reader, capped at 75ch |

The main metric and medium-weight content retain their emphasis. Section headings no
longer compete with the 16px bold insight. The signal and management headings have 16px
separation from their content, with no extra metric top margin inside the signal.
Major regions retain their 24px gap. Small text uses neutral tracking; the insight/view
title uses -0.01em, detail/dialog titles -0.015em, and the metric -0.025em.

`HomeBrandText` applies `hc-inline-brand` to its pills so the font size, weight,
line-height, and tracking inherit from the sentence or title. Structured comparison,
leader, engine-selector, and menu pills use 13px; competitor-management names inherit
their 14px content role. Metadata pills retain their compact 12px labels. Logo sizes,
pill surfaces, wrapping, and icon-only names are unchanged.

Scoped typography covers controls on the overview and prompt detail as well as editor,
confirmation, select, and combobox portals. Input text remains 16px at widths up to
640px. Existing control dimensions, heading semantics, focus styles, responsive layout,
motion, copy, fixtures, and state behavior are retained. Production primitive defaults
and the application's Inter font are unchanged.

Verification commands: `make format`, `pnpm --dir ui check`, `pnpm --dir ui check-types`,
and `pnpm --dir ui test`. Browser checks remain assigned to the user: compare hierarchy
with Fey at the same viewport; open prompt detail, editors, filters, competitor management,
and Settings; inspect long questions and brand names at desktop, 390px, 320px, and 200%
zoom. Verify keyboard focus and return navigation, plus loading, empty, error, and
missing-comparison states. Source checks do not establish rendered font selection,
contrast, wrapping, or visual fidelity.
