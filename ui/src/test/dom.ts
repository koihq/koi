import { expect } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

/** Waits for the first element matching `selector` and returns a locator for it. */
export async function css(selector: string): Promise<Locator> {
  await expect.poll(() => document.querySelector(selector), { timeout: 5_000 }).toBeTruthy();

  return page.elementLocator(document.querySelector(selector) as Element);
}

export function box(locator: Locator) {
  return locator.element().getBoundingClientRect();
}

/** The opacity the element is drawn with: its own multiplied by every ancestor's. */
export function opacity(element: Element | null) {
  if (!element) return null;

  let value = 1;
  for (let node: Element | null = element; node; node = node.parentElement) {
    value *= Number(getComputedStyle(node).opacity);
  }

  return value;
}

/**
 * The text a reader sees, including open shadow roots where components like NumberFlow draw
 * digits. Styles and inert copies (NumberFlow's off-screen digits) are skipped.
 */
export function deepText(node: Node): string {
  if (node instanceof HTMLStyleElement || (node instanceof Element && node.hasAttribute("inert"))) {
    return "";
  }

  const own = node instanceof Element && node.shadowRoot ? deepText(node.shadowRoot) : "";
  const children = [...node.childNodes]
    .map((child) =>
      child.nodeType === Node.TEXT_NODE ? (child.textContent ?? "") : deepText(child),
    )
    .join("");

  return own || children;
}

/** Moves the pointer to the top-left corner of the page, away from any hover target. */
export function pointerAway() {
  return userEvent.hover(page.elementLocator(document.body), { position: { x: 1, y: 1 } });
}

/** True when the page does not scroll sideways. */
export function fitsViewport() {
  return document.documentElement.scrollWidth <= window.innerWidth;
}

/** A promise the test resolves to let held requests continue. */
export function gate() {
  let release!: () => void;
  const opened = new Promise<void>((resolve) => {
    release = resolve;
  });

  return { opened, release };
}
