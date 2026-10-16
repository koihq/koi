# Tailwind and Inter cleanup

- **Status:** Ready. The user chose Inter Variable for the app.
- **Baseline:** `f9b0153` plus the uncommitted home-context migration. Keep its single-brand layout and API behavior.
- **Reference:** `/dev/explorations/home-context` remains the visual specification. This revises `plans/ui/home-context-fidelity.md`, so its inspected Mobbin research is reused: [Fey portfolio](https://mobbin.com/screens/f4b1fb8e-1db1-407c-a1b1-41ec1eb312c7) for compact comparisons, [Fey home](https://mobbin.com/screens/64282196-4e6c-4590-a0d1-07bde4e4819b) for the leading signal, and [Peec AI](https://mobbin.com/screens/0711fee8-0de2-4f0e-8b7c-16387d22f091) for evidence labels. No new visual pattern is introduced.

## Design principles

The founder needs the same reading order while the implementation gets simpler: keep the signal, prompt management, detail evidence, and competitor list in their current positions and sizes. Put ordinary geometry, type, hover, selected, and breakpoint rules on the element they style. Keep CSS for global semantic tokens, font-face declarations, and primitive motion. This makes each component reviewable without a long parallel stylesheet.

Use one global `--font-sans` stack with the installed Inter Variable package. Remove Calibre declarations from the production workspace and exploration so font selection is consistent. No font dependency change is needed.

## Skill checks

| Rule | Decision | Verification |
| --- | --- | --- |
| Product design / frontend design | Preserve the home-context composition and type roles. | Desktop and mobile render comparison. |
| Codebase patterns / ponytail | Use existing Tailwind and shadcn; no new styling library or abstraction. | Build and class audit. |
| Typography / accessibility | One font stack, readable fallback, unchanged semantics and focus states. | Computed font, keyboard, and zoom checks. |
| Layout / colors / UI | Preserve semantic tokens and breakpoints while moving declarations. | Light/dark and 390px/1440px checks. |
| React practices | Keep data fetching and component behavior untouched. | Typecheck and tests. |

## Handoff

1. In `ui/src/styles.css`, keep Tailwind and Inter imports, semantic theme tokens, base body font, and shared primitive motion. Remove component layout selectors after moving their declarations to JSX. Retain CSS only where Tailwind cannot express a shared rule cleanly.
2. In Home, prompt manager, visibility chart, prompt detail, and competitor management, replace `koi-*` presentation classes with equivalent Tailwind utilities at the same breakpoints. Keep state selectors (`aria-pressed`, `data-selected`, focus) and accessible names.
3. Set `--font-sans` to Inter Variable for all routes, remove local Calibre declarations, and update design docs and the UI catalog.
4. Check `pnpm --dir ui check`, `check-types`, `test`, `build`, and visually compare desktop/mobile light/dark screens with the earlier screenshots. Verify no server is left on port 5173.

## Acceptance

- The current product screens retain their geometry and interaction states.
- Most route and block presentation rules are Tailwind utilities, and `styles.css` no longer hosts a second component styling system.
- Inter Variable is the only app font. No Calibre declarations remain in active UI CSS.
