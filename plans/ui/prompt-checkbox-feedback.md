# Prompt table checkbox feedback

**Status:** Ready. **Target:** `/dev/explorations/home-context` prompt table and mobile list. **Baseline:** `5046405`, with the prompt table redesign uncommitted. This change covers selection controls only; selection state and fixture data stay as implemented.

## User job and research

The founder selects prompts for bulk confirm or delete. The current native 24px checkbox looks oversized beside the compact table and changes state without visual feedback.

Mobbin `search_screens` queries covered Fey's selectable financial table, Linear's issue list, and Shopify's order table. [Fey](https://mobbin.com/screens/f3237cad-944d-4fc4-b330-7c94fac3ef60) shows small checkboxes in a dense data grid. [Shopify](https://mobbin.com/screens/fb87b68d-8192-495b-913b-e103e73aa7bb) keeps selection boxes visually quiet at the left edge of an order table. The [Linear result](https://mobbin.com/screens/fd1b4d88-f021-49a3-98af-4cd3a87e1d29) supports the compact row scale but does not reveal selection animation. Static images cannot establish motion or keyboard behavior. Use the user's screenshot for the actual Koi defect.

## Direction and skill preflight

The visible control should be 16px while its target remains 28px. Keep native checkbox semantics and the existing selection owner in `PromptList`; add only a decorative visual layer. The check mark should appear in 140ms for pointer interaction, as short state feedback. Keyboard activation changes instantly; reduced motion omits the transform. Keep selected state visible without motion. This retains Koi's quiet table and avoids a new library or shared primitive.

| Skill/rule | Decision | Verification |
| --- | --- | --- |
| product-design, codebase-patterns: scoped exploration and existing controls | Change only this route's checkbox presentation | Both themes in Vite |
| better-accessibility: native semantics, visible focus, 24px target | Input remains focusable and 28px; decorative layer ignores pointer events | Keyboard and touch |
| emil-design-eng, animate, better-ui: high-frequency feedback is short and interruptible | CSS transition on check opacity/transform, no keyframe or page motion | Rapid select/deselect |
| ponytail: reuse native state | No new dependency or selection logic | Existing selection tests |

## Implementation handoff

1. In `ui/src/routes/dev/explorations/-components/prompt-list.tsx`, wrap each of the four existing checkboxes in a 28px container and append an `aria-hidden` visual span after the input. Preserve refs, checked values, labels, `onChange`, and the mobile wrapping label.
2. In `ui/src/routes/dev/explorations/-home-context.css`, make the input cover the 28px wrapper invisibly. Draw a 16px box in the decorative span; use `:checked` and `:indeterminate` for fill and glyph. Move desktop hover concealment to the wrapper. Style `input:focus-visible + span` with the existing ring token. Keep the visible glyph in forced colors.
3. Use 140ms `var(--ease-out)` opacity/transform transition only when `prefers-reduced-motion: no-preference` and the input is not `:focus-visible`. The visual checked state is immediate otherwise.

No API, locale, loading, or error behavior changes. Unselected, selected, and indeterminate states must all be visible. On desktop, an unselected row checkbox appears on hover or focus; selected rows keep it visible. On mobile it remains visible. Header select-all must show an indeterminate dash for partial selection.

Verify `pnpm --dir ui check`, `pnpm --dir ui check-types`, and `pnpm --dir ui test`. At `http://localhost:5173`, inspect dark and light, keyboard Tab/Space, pointer selection, rapid toggle, reduced motion, mobile width, and 200% zoom. No browser verification has been completed for this handoff.
