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
files have been added. The current Home study tries locally installed Calibre Regular and
Medium, then Helvetica Neue, Helvetica, and Arial. On a Mac without Calibre this renders
Helvetica Neue. This is an approximation; exact parity needs licensed Calibre webfont
assets and verification against the original app. Other Koi routes still use Inter.

## Surface system

| Role | Working value | Use |
| --- | --- | --- |
| Canvas | `#121316` | Workspace |
| Content | `#0c0c10` with a very faint diagonal highlight | Chart and answer tiles |
| Recap | `#08090b` with a low-opacity warm highlight | Narrative recap |
| Glass | `#252529c9` plus a translucent white highlight | Dock, search, answer panel, brand menu |
| Primary text | `#e4e4e8` | Readable content |
| Secondary text | `#a4a4ad` | Metadata |
| Comparison | Desaturated lavender | Alternative chart line |
| Outcome | Muted sage / rose | Mention outcomes |

Glass uses 24–32px backdrop blur, a subtle inset top highlight, and a soft shadow on
floating surfaces. Content uses deeper surfaces with faint edge illumination. These
are reproduction choices from the screenshots, not recovered Fey CSS values.

## Typography and spacing

The previous 9–12px-heavy implementation was too small. Current roles:
- Greeting: 18px medium; section headings: 15px medium.
- Recap: 16px with 1.6 line height; answer body: 15px with 1.55 line height.
- Rows and controls: 13–14px; metadata: 12px.
- Visibility rate: 42px with tabular figures and restrained negative tracking.

Body tracking is neutral. Layout is capped at 1320px with a 24px column gap and
24–28px surface padding. Content corners are 10px; floating panels are 12px; the dock
is a capsule. At narrow widths the columns stack. No new motion is introduced.

## Additional Mobbin references examined

- [Portfolio](https://mobbin.com/screens/ba4a167d-2b95-49dd-b8d9-fd8ddbeb90dd):
  inspected at full resolution for type proportions, control highlights and dock material.
- [Search overlay](https://mobbin.com/screens/05f87eae-99d3-4d7a-a7c4-c443a99afb0b):
  glass search surface floating over the dashboard.
- [Context panel](https://mobbin.com/screens/8487cb92-3969-4f21-a847-26177466cf9a):
  a right-hand inspection surface keeps the overview in context.

Koi adapts these to brand comparison, engine coverage, answer search, answer inspection,
and an agent brief. The search button is a separate glass circle beside the dock.

## Home revision: Fey overview composition

Open `/dev/explorations/home-context` on the Vite server. The workspace banner,
large goal editor, tracking-scope form, exploration bar, and session footer were removed
at the user's request. This replaces the earlier header-first study.

The direct composition reference is [Fey's overview](https://mobbin.com/screens/dc5328bd-98d9-46b1-9b18-7ee5805fb878):
- A compact brand-avatar switcher and a quiet demo-data indicator.
- A narrower left surface holding the chart, inline period controls, and compact ranking.
- A wider right column of separate dark recap and answer tiles.
- A small floating dock. Its controls jump to Home, comparison, answers, and brief;
  the final control returns to the exploration gallery. This is a navigation experiment,
  not a decision on the production shell.

Chart lines compare the selected brand and its first tracked alternative on the same
0–100% scale. Ranking and recap use the same filtered synthetic answer records.
Brand switching restores each brand's selected period. Buyer-question scope defaults
to alternatives for Deja and discovery for Koi; all engines are included.
Goals remain in the demo brief but are not shown or editable in this iteration.

No real collection or agent connection is represented. Evidence and answer details are
labeled as demo data; the fixture period ends October 13, 2026. The font stack and its
remaining fidelity limitation are documented above.

Manual checks for the user:

1. Switch brands through the avatars. Check names, comparison, and answers change together.
2. Select 7D, 30D, and 90D. Check the timeline, sample count, ranking, and recap update.
3. Open an answer, then close with Escape. Check focus returns to the selected tile.
4. Download a brief for each brand; verify the brand and selected period.
5. Use the dock to reach each section and return to the gallery.
6. Check 390px and 320px widths for overflow and dock overlap with the last action.
7. Open search, filter by a question or engine, open a result, and close with Escape.
8. Compare the desktop layout against the linked Fey overview, particularly column
   proportions, compact type, chart line weight, tile spacing, and dark surface contrast.

No browser testing is performed, per the user's instruction.


## Brand avatar switcher

The greeting and Koi workspace mark have been removed. Brand initials occupy 26px
buttons with 24px colored discs. The collapsed centers are 20px apart, creating a
small overlap with a 1px canvas-colored rim. Active avatars have full opacity and a
thin ring; inactive avatars retain 55% opacity. Colors are assigned consistently from
a mixed violet, teal, amber, blue, rose, and olive palette, including menu entries.

Hovering the group spreads its centers to 32px apart using interruptible 200ms
transform transitions. The group reserves its expanded width so neither neighboring
content nor the hover boundary shifts. Leaving it restores the stack. Keyboard focus
and an open overflow menu keep it expanded; keyboard and reduced-motion changes are
immediate. Touch/coarse-pointer devices show the separated row. Names appear in
tooltips on hover and focus.

The configurable `MAX_VISIBLE_BRANDS` in `home-context.tsx` is 3. Additional brands go
into a +N menu. Selecting an overflow brand places it in the final visible slot;
the displaced brand becomes available in the menu. Existing slots retain their order.
The demo includes Deja, Koi, Orbit, Sprout, Forma, and Northstar. The last four are
fictional workspace examples with separate question sets, alternatives, and goals.
Three avatars and a +3 menu are visible immediately. Select an overflow brand and
check that its own chart, answers, search results, and exported brief are shown.
Period selections remain independent for all six brands.

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
