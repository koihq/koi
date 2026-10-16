---
name: product-design
description: Koi visual judgment — founder/AEO reader job, composition, PNG mark, stillness, and named anti-patterns. Use before any visual or UI work (routes, blocks, tokens, primitives). Screen and flow reviews go through better-interface; branch, PR, or uncommitted UI diffs go through interface-review.
---

# Product design

Koi is an agent-first AEO tool. A founder opens it to answer: do answer engines mention us, who do they recommend instead, and what should my agent do next.

Read [primitives.md](primitives.md) for tokens and the shadcn inventory. Mechanics live in `ui/src/styles.css` and `ui/src/components/ui`. Do not invent a parallel type scale, palette, or component set.

This file owns Koi judgment. Accessibility, layout, writing, type, color, and polish rules live in the installed `better-*` skills — point at them, do not copy them here.

## Reader

Two speeds on every product surface:

1. **Executive** — mention, top substitutes, the brief. Visible in the first viewport.
2. **Audit** — prompt × engine × citation. Full width when it is a table.

Frame the screen around that job. Do not start from a dashboard template.

## Composition

Use this geometry when a mentions surface exists:

```
rail |  brand + verdict + engines     |  runs / citations / brief
     |  (facts, definition list)      |  (activity, not a widget grid)
```

- The authenticated shell has a labeled sidebar that collapses to a narrow icon rail. The main area is an inset surface on desktop.
- **Definition-list metadata:** muted label column, value column. Gap and type role group rows. No boxes, no rules between rows.
- The title is the page. A run id stays quiet (`tabular-nums`). The sentence is the heading.
- One focal relationship per viewport. Magnitude shares one scale; comparison uses the same basis; the markdown brief is one primary object.

Auth is a quiet centered column. Home stays a brief introduction until there is real mention data.

Light and dark are the same layout. Invert the canvas, keep the hierarchy. No separate “marketing light” and “tool dark.”

## Copy

Keys live in `ui/src/locales`. Sentence case. CTAs name the action (`Sign in`, not `Submit`). Errors say how to recover. Empty states invite an action. No em dashes as a style tic. No authoring vocabulary in UI copy (“composition”, “hierarchy”).

**Let context do the explaining.** Do not repeat information already established by
navigation, selection, or nearby controls. Remove explanatory text that adds no useful
meaning. Prefer a familiar icon when its meaning is clear, with an accessible name and
a tooltip when needed. Put secondary definitions and evidence behind contextual
disclosure. Keep text that identifies an ambiguous metric, explains a consequential
state, or helps someone recover.

For example, do not repeat the selected brand beside its switcher, add “All tracked
prompts” when the view already establishes that scope, restate the selected period, or
print chart interaction instructions beneath an inspectable chart. Preserve accessible
names and keyboard behavior when removing visible instructions.

## PNG mark

The mark is the one loud thing. Orange is identity, not the theme.

The asset is a PNG copied as-is (`ui/src/assets/koi-mark.png`). It has a transparent field. Do not flatten it onto black, vectorize, crop, recolor, invert, or invent a white-on-orange lockup. Same file on the light canvas and the dark canvas. Copy from the source PNG — chat previews may be JPEGs with the alpha baked to black.

`--brand` is sampled from that orange. `--primary` stays Mira near-black / near-white. Do not fill primary buttons orange. Do not set a full-app orange canvas.

## Motion

Daily tool: default is stillness. Encode motion only when you can name the purpose as **feedback**, **spatial consistency**, or **state indication**. Can’t name it? Don’t animate.

What this product encodes today:

- **Button press** — feedback that the control heard the pointer. `:active` `scale(0.97)` at **160ms** `var(--ease-out)`. Keep the `not-aria-[haspopup]` exception so menus do not shrink. Color/hover uses `ease`. No hover **scale**. Gate any future hover transform with `@media (hover: hover) and (pointer: fine)`.
- **Input** — focus/invalid is state indication. Color transitions only. No press scale.
- **Label, Field, Separator** — none.

Named CSS properties only (`transform`, `background-color`, `color`, `border-color`, `box-shadow`, `opacity`). Never `transition: all`. Never `translateY` as a stand-in for press.

Do not animate theme flips, keyboard-initiated actions, or page navigation. Do not add popover, tooltip, or modal recipes until those primitives exist. No Motion/Framer.

`prefers-reduced-motion: reduce` drops **transform** motion and keeps color/opacity. Reduced motion means no scale on `:active`.

Curves and durations are the two tokens in [primitives.md](primitives.md). Implement from `emil-design-eng` and `animate` (`RECIPES.md` button press). Do not invent a third curve. `better-ui` owns polish/motion review through `better-interface`.

## Anti-patterns

These are disallowed, not taste. Name them when rejecting work.

| Name | Never | Instead |
| --- | --- | --- |
| **Mono chrome** | `--font-mono`, `font-mono`, Geist Mono, JetBrains, mono on IDs, timestamps, KPIs, tables, “code-like” labels | Inter Variable for everything. Aligned numbers use `tabular-nums` |
| **Card kit** | shadcn `Card`; boxed metrics, metadata, sections, or lists “to group them” | Gap, alignment, type role. A surface is earned (dialog, popover, a quoted artifact) |
| **Hairline habit** | `Separator` / `FieldSeparator` between fields, under headings, between timeline rows | Space owns grouping. One structural split at most (rail \| body, facts \| activity) |
| **Decorative shadow** | Shadows on static panels or sections, including the authenticated main surface | Use canvas contrast and a single structural border. Add a shadow only when it clarifies meaningful overlap or elevation |
| **Primitive override** | Route `className` changing size or type of a `ui/` control (`h-9 text-sm` on `Input`) | Use the primitive’s CVA sizes. Density is Mira compact |
| **Parallel theme** | Page-local hex, a second typeface, a new radius, orange `--primary` | Semantic tokens and `components/ui`. New controls: `pnpm --dir ui exec shadcn add …`, then update [primitives.md](primitives.md) |

## Reviews

- Screen or flow → `better-interface` (it already routes to `better-accessibility`, `better-layout`, `better-writing`, `better-typography`, `better-colors`, `better-ui`).
- Branch, PR, or uncommitted UI diff → `interface-review`.
