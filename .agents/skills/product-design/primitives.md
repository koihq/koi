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

Type: Inter Variable (`--font-sans`) only. **Never add `--font-mono` or `font-mono`.** Tabular numbers: `tabular-nums`.

Type roles already in screens:

| Role | Classes |
| --- | --- |
| Page title | `text-[30px] font-semibold tracking-[-0.035em]` |
| Section title | `text-lg font-medium tracking-tight` |
| Main metric | `text-[38px] font-medium tabular-nums` |
| Body | inherit `foreground` |
| Supporting | `text-muted-foreground` |
| Control / label | primitive default (`text-xs/relaxed`, `FieldLabel`) |
| Changing digits | `tabular-nums` |

Icons: Hugeicons only (`components.json` `iconLibrary`). No mixed sets. No icons in colored tiles.

## shadcn inventory

Routes and blocks import `@/components/ui/*`. Base UI stays inside those files.

| Need | Use |
| --- | --- |
| Action | `Button` variants `default \| outline \| secondary \| ghost \| destructive \| link`; sizes `xs \| sm \| default \| lg \| icon \| icon-xs \| icon-sm \| icon-lg` |
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

`FieldGroup` gap is the default grouping. Do not add `Card`, `Table`, or `Tabs` until a product screen needs the behavior.

Authenticated navigation uses `AppHeader`: one brand link, competitors, account menu, theme, and sign out. The old Sidebar remains in the repository only for unused legacy code until cleanup.

## Authenticated shell baseline

The compact header and main share a 1320px max width. Side padding is 16px on mobile and 28px from `sm`. Header actions use 40px hit targets. The brand icon is 28px. A single structural border around the signal separates it from the canvas. Keep focus rings separate from borders.

## Motion contract

Named properties only: `transform`, `background-color`, `color`, `border-color`, `box-shadow`, `opacity`. Never `all`. Never reintroduce `transition-all` or `active:translate-y-px` on `Button`.

Tokens live in `ui/src/styles.css`. Button press is on `[data-slot="button"]` — do not add `transition-*` or `active:*` motion classes in `button.tsx`.

`prefers-reduced-motion: reduce` drops **transform** (and the `:active` scale) and keeps color/opacity. Wrap press transform in `prefers-reduced-motion: no-preference` so it is opt-in.

| Primitive | Motion |
| --- | --- |
| `Button` | Press: `:active` `scale(0.97)` at **160ms** `var(--ease-out)`, keep `:not([aria-haspopup])`. Color/hover: **160ms** `ease`, no hover scale. Reduced motion: no scale on `:active` |
| `Input` | `transition-colors ease` only. No press scale |
| `Label`, `Field`, `Separator` | None |

Do not animate theme flips, keyboard actions, or page navigation. Do not add popover/tooltip/modal recipes until those primitives exist.
