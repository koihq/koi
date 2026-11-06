import { type RefObject, useLayoutEffect, useRef } from "react";

/**
 * Animates children of `container` (marked with `data-flip-key`) from their previous
 * position when the list order changes, so a re-sorted list moves instead of jumping.
 * Uses transform only and stays still under reduced motion.
 */
export function useFlip(container: RefObject<HTMLElement | null>, order: string) {
  const positions = useRef(new Map<string, number>());

  // biome-ignore lint/correctness/useExhaustiveDependencies: `order` is the trigger; positions are read from the DOM.
  useLayoutEffect(() => {
    const root = container.current;
    if (!root) return;

    const items = [...root.querySelectorAll<HTMLElement>("[data-flip-key]")];
    const previous = positions.current;
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const next = new Map<string, number>();

    for (const item of items) {
      const key = item.dataset.flipKey ?? "";
      const top = item.offsetTop;
      const before = previous.get(key);

      next.set(key, top);
      if (reduced || before === undefined || before === top) continue;

      item.animate([{ transform: `translateY(${before - top}px)` }, { transform: "none" }], {
        duration: 260,
        easing: "cubic-bezier(0.23, 1, 0.32, 1)",
      });
    }

    positions.current = next;
  }, [container, order]);
}
