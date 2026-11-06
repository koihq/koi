# Primitives

The names agents may use. Do not read `styles.css` into context. Do not hand-roll a control that exists in shadcn.

After `pnpm --dir ui exec shadcn add …`, append the new primitive here in the same PR.

## Tokens

| Role | Token | Notes |
| --- | --- | --- |
| Page canvas | `background` / `foreground` | Light `:root` is the default. `.dark` is first-class |
| Signal and answer surface | `surface` / `surface-foreground` | Main signal, saved answer, and forms |
| Quiet text | `muted` / `muted-foreground` | Labels in a definition list, timestamps |
| Controls | `primary` / `primary-foreground` | Near-black / near-white. Not orange |
| Mark / scarce accent | `brand` | Logo orange only. Not `--primary` |
| Danger | `destructive` | Error text and invalid rings |
| Changes | `positive` / `negative` | Directional metric changes |
| Edges | `border`, `input`, `ring` | Structure and focus, not decoration |
| Radius | `--radius` and `rounded-sm` … `rounded-4xl` | Mira scale. Do not pick a one-off `rounded-*` |
| Motion out | `--ease-out` | `cubic-bezier(0.23, 1, 0.32, 1)` |
| Motion on-screen | `--ease-in-out` | `cubic-bezier(0.77, 0, 0.175, 1)` |

Type: Hanken Grotesk Variable (`--font-sans`, `@fontsource-variable/hanken-grotesk`), the open stand-in for Fey's Calibre. One typeface only. **Never add `--font-mono` or `font-mono`.** Tabular numbers: `tabular-nums`.

Type roles are tokens in `styles.css` `@theme`. Never write `text-[13px]`-style sizes in routes. One strong weight: `font-semibold`; `font-bold` is not used.

| Role | Classes |
| --- | --- |
| Page title | `text-title font-semibold` (30px, -0.03em) |
| Section title | `text-section font-semibold` (16px) |
| Main metric | `text-metric font-medium tabular-nums` (40px) |
| Body / table cell | `text-body` (14px) |
| Supporting | `text-muted-foreground` |
| Control / label | primitive default (`text-xs/relaxed`, `FieldLabel`) |
| Changing digits | `tabular-nums` |

Icons: Hugeicons only (`components.json` `iconLibrary`). No mixed sets. No icons in colored tiles.

## shadcn inventory

Routes and blocks import `@/components/ui/*`. Base UI stays inside those files.

| Need | Use |
| --- | --- |
| Action | `Button` variants `default \| outline \| secondary \| ghost \| destructive \| link`; sizes `xs \| sm \| default \| lg \| icon \| icon-xs \| icon-sm \| icon-lg \| circle \| circle-sm \| pill` |
| Checkbox | `Checkbox` with checked/indeterminate states; 16px box and 28px hit target |
| Text field | `Input` + `Field` / `FieldLabel` / `FieldError` / `FieldGroup` |
| Searchable choice | `Combobox` / `ComboboxInput` / `ComboboxContent` / `ComboboxItem` |
| Action menu | `DropdownMenu` / `DropdownMenuTrigger` / `DropdownMenuContent` / `DropdownMenuItem` |
| Data table | `Table` / `TableHeader` / `TableBody` / `TableRow` / `TableHead` / `TableCell` |
| Modal form | `Dialog` / `DialogContent` / `DialogHeader` / `DialogTitle` / `DialogDescription` |
| Destructive confirmation | `AlertDialog` / `AlertDialogContent` / `AlertDialogAction` / `AlertDialogCancel` |
| Compact choice | `Select` / `SelectTrigger` / `SelectValue` / `SelectContent` / `SelectItem` |
| Label | `Label` (prefer `FieldLabel` in forms) |
| Structural split | `Separator` — **one split**, never as grouping. Do not put `FieldSeparator` between form fields |
| Data chart | `ChartContainer` / `ChartTooltip` with Recharts series |

`FieldGroup` gap is the default grouping. Do not add `Card` for grouping. Product data tables use the installed `Table` primitives through `DataTable`.

Shared product blocks and utilities:

| Need | Use |
| --- | --- |
| Interactive list table | `DataTable interactive`: no rules, one rounded `signal-pill` fill per row on hover, focus and selection; siblings recede to 0.55 |
| Row list (mobile, run history, rivals) | `ul.list-rows` with row buttons using `hover:bg-signal-pill`; same sibling fade |
| Floating surface (bulk bar, hover previews, metric tooltip) | `surface-float` utility; never re-type the glass recipe |
| Engine identity | `EngineIcon` (monochrome Hugeicons). Never draw engines with favicons |
| Engine outcome | `OutcomeMark` (filled = cited, tinted = mentioned, outlined = not in answer) with `OutcomeLegend` |
| Dates, rates, deltas | `lib/format.ts`: `formatDay`, `formatDateTime`, `formatDayRange`, `formatRate`, `formatPoints` ("−16.0 pp") |
| Chart period and engine filter | `RangeControl`, `EngineFilter` (`chart-controls`): one row beside the chart title, on every chart |
| Amount picker | `TickPicker` (`tick-picker`): large value over a tick ruler, for a few uneven amounts. Plain props, reusable outside the app |
| Row selection | A controls-only column left of the text, as in Linear: the checkbox hangs in the panel's 44px side padding, with 12px on each side of it, so row text stays aligned with the panel title |
| Way back | `BackLink` (`back-link`): arrow icon only, named by `aria-label` |
| Add to a list | `icon-outline` `circle-sm` button with `Add01Icon`, beside the list's search |
| Re-sorting lists | `useFlip` so rows move instead of jumping |

Authenticated navigation uses `AppHeader`: one brand link, competitors, account menu, theme, and sign out. There is no sidebar; the legacy Sidebar and Sheet were removed.

## Authenticated shell baseline

The compact header and main share a 1376px outer max width (1320px content after desktop padding). Side padding is 16px on mobile and 28px from `sm`. Header actions use 40px hit targets. The brand icon remains compact. A single structural border around the signal separates it from the canvas. Keep focus rings separate from borders.

## Motion contract

Named properties only: `transform`, `background-color`, `color`, `border-color`, `box-shadow`, `opacity`. Never `all`. Never reintroduce `transition-all` or `active:translate-y-px` on `Button`.

Tokens live in `ui/src/styles.css`. Button press is on `[data-slot="button"]` — do not add `transition-*` or `active:*` motion classes in `button.tsx`.

`prefers-reduced-motion: reduce` drops **transform** (and the `:active` scale) and keeps color/opacity. Wrap press transform in `prefers-reduced-motion: no-preference` so it is opt-in.

| Primitive | Motion |
| --- | --- |
| `Button` | Press: `:active` `scale(0.97)` at **160ms** `var(--ease-out)`, keep `:not([aria-haspopup]):not(:focus-visible)`. Color/hover: **160ms** `ease`, no hover scale. Reduced motion: no scale on `:active` |
| `Input` | `transition-colors ease` only. No press scale |
| `Label`, `Field`, `Separator` | None |

Do not animate theme flips, keyboard actions, or page navigation. Do not add popover/tooltip/modal recipes until those primitives exist.

## Current production decisions

See `docs/design-system.md` for the current contract and historical exploration provenance.
Competitors show name, domain, visibility and gap vs the brand. Checkbox uses shadcn with a 28px hit target.
Input and Textarea use 16px below `sm`; preserve their desktop density. Button `circle`
and `pill` sizes provide 40px targets without route-level size overrides. Tables with
overflow use a localized hint and keyboard-focusable scroll container.

Header account/competitor controls use `circle-sm`: 32px visible, 40px hit target.
Account uses `icon-outline` for a transparent idle background in both themes.
Summary owns the shared visibility-chart style, including detail and boundary values.
