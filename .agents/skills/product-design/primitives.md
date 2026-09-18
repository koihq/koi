# Primitives

The names agents may use. Do not read `styles.css` into context. Do not hand-roll a control that exists in shadcn.

After `pnpm --dir ui exec shadcn add …`, append the new primitive here in the same PR.

## Tokens

| Role | Token | Notes |
| --- | --- | --- |
| Page canvas | `background` / `foreground` | Light `:root` is the default. `.dark` is first-class |
| Quiet text | `muted` / `muted-foreground` | Labels in a definition list, timestamps |
| Controls | `primary` / `primary-foreground` | Near-black / near-white. Not orange |
| Mark / scarce accent | `brand` | Logo orange only. Not `--primary` |
| Danger | `destructive` | Error text and invalid rings |
| Edges | `border`, `input`, `ring` | Structure and focus, not decoration |
| Radius | `--radius` and `rounded-sm` … `rounded-4xl` | Mira scale. Do not pick a one-off `rounded-*` |
| Motion out | `--ease-out` | `cubic-bezier(0.23, 1, 0.32, 1)` |
| Motion on-screen | `--ease-in-out` | `cubic-bezier(0.77, 0, 0.175, 1)` |

Type: Inter Variable (`--font-sans`) only. **Never add `--font-mono` or `font-mono`.** Tabular numbers: `tabular-nums`.

Type roles already in screens:

| Role | Classes |
| --- | --- |
| Page title | `text-2xl font-medium tracking-tight` |
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
| Label | `Label` (prefer `FieldLabel` in forms) |
| Structural split | `Separator` — **one split**, never as grouping. Do not put `FieldSeparator` between form fields |

`FieldGroup` gap is the default grouping. Do not add `Card`, `Table`, or `Tabs` until a product screen needs the behavior.

Authenticated navigation uses shadcn `Sidebar` with `SidebarProvider`, `SidebarInset`, `SidebarTrigger`, `SidebarMenuButton`, and their generated `Sheet`, `Tooltip`, and `Skeleton` dependencies. Desktop collapses to icons; mobile opens a drawer.

## Authenticated shell baseline

These are the current measurements at a 16px root font size. Keep them consistent when extending the shell; change them through the shared component or token when the product needs a different scale.

| Element | Current value |
| --- | --- |
| Main inset edge | 0.5px `border-border` from `md` (768px) upward; no border on mobile; no shadow |
| Main inset corner | `rounded-xl` = 14px (`--radius: 10px` × 1.4) on desktop |
| Sidebar link corner | 8px (`--radius-sm: 6px` + 2px) |
| Page title in shell header | `text-sm font-medium` = 14px / 500 |
| Organization name | 13px / 500 |
| Sidebar link label | 13px / 500, including active links |
| Sidebar width | 232px expanded; 48px icon rail |
| Sidebar states | Default transparent, hover `sidebar-accent`, selected `sidebar-selected` |
| Placeholder body text | Browser body default = 16px / 400 |
| Navigation icon | 16 × 16px, Hugeicons stroke width 2 |
| Sidebar toggle icon | 16 × 16px, Hugeicons stroke width 2 |
| Koi mark in sidebar | 20 × 20px |

The border is a structural edge, not a cue to add more dividers. Focus rings are separate from borders. Use the token radius scale; do not introduce a route-specific corner value.

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
