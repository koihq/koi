import { describe, expect, it } from "vitest";

import { classifyLogo, dominantColor } from "@/lib/logo-shape";

const size = 64;

/** Builds RGBA data whose alpha comes from `alpha(x, y)`. */
function image(alpha: (x: number, y: number) => number) {
  const data = new Uint8ClampedArray(size * size * 4);

  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      data[(y * size + x) * 4 + 3] = alpha(x, y);
    }
  }

  return data;
}

const center = (size - 1) / 2;
const insideCircle = (x: number, y: number, radius: number) =>
  (x - center) ** 2 + (y - center) ** 2 <= radius ** 2;

describe("classifyLogo", () => {
  it("fills assets with an opaque background", () => {
    expect(
      classifyLogo(
        image(() => 255),
        size,
        size,
      ),
    ).toBe("filled");
  });

  it("fills assets that are already a round badge", () => {
    expect(
      classifyLogo(
        image((x, y) => (insideCircle(x, y, 32) ? 255 : 0)),
        size,
        size,
      ),
    ).toBe("badge");
  });

  it("treats a transparent glyph as a symbol", () => {
    expect(
      classifyLogo(
        image((x, y) => (insideCircle(x, y, 18) ? 255 : 0)),
        size,
        size,
      ),
    ).toBe("symbol");
  });

  it("treats a faint background as a symbol", () => {
    expect(
      classifyLogo(
        image(() => 90),
        size,
        size,
      ),
    ).toBe("symbol");
  });

  it("falls back to a symbol for tiny images", () => {
    expect(classifyLogo(new Uint8ClampedArray(4 * 4 * 4), 2, 2)).toBe("symbol");
  });
});

describe("dominantColor", () => {
  function solid(rgb: [number, number, number], share = 1) {
    const data = new Uint8ClampedArray(size * size * 4);

    for (let index = 0; index < size * size; index++) {
      const colored = index < size * size * share;
      data.set(colored ? [...rgb, 255] : [0, 0, 0, 255], index * 4);
    }

    return data;
  }

  it("returns the brand hue of a colorful mark", () => {
    expect(dominantColor(solid([79, 70, 229]), size, size)).toBe("#4f46e5");
  });

  it("returns null for monochrome marks", () => {
    expect(dominantColor(solid([0, 0, 0]), size, size)).toBeNull();
    expect(dominantColor(solid([255, 255, 255]), size, size)).toBeNull();
    expect(dominantColor(solid([128, 128, 128]), size, size)).toBeNull();
  });

  it("ignores a color that covers only a speck of the mark", () => {
    expect(dominantColor(solid([220, 38, 38], 0.03), size, size)).toBeNull();
  });
});
