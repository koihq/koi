/**
 * How a square logo asset should sit in a round icon, read from its own pixels.
 *
 * - `filled`: the asset has an opaque background; crop it to the circle.
 * - `badge`: the asset is already a round mark (transparent corners, opaque edges); fill.
 * - `symbol`: a transparent glyph; place it smaller on a disc.
 */
export type LogoShape = "filled" | "badge" | "symbol";

const OPAQUE = 200;
const CLEAR = 40;

function alphaAt(data: ArrayLike<number>, width: number, x: number, y: number) {
  return data[(y * width + x) * 4 + 3];
}

function share(values: number[], test: (alpha: number) => boolean) {
  return values.length ? values.filter(test).length / values.length : 0;
}

/** Classifies RGBA pixel data (as from `getImageData`) by the alpha of its border. */
export function classifyLogo(data: ArrayLike<number>, width: number, height: number): LogoShape {
  if (width < 4 || height < 4) return "symbol";

  const ring: number[] = [];
  const band = Math.max(1, Math.round(Math.min(width, height) / 32));

  for (let x = 0; x < width; x++) {
    for (let offset = 0; offset < band; offset++) {
      ring.push(alphaAt(data, width, x, offset), alphaAt(data, width, x, height - 1 - offset));
    }
  }
  for (let y = band; y < height - band; y++) {
    for (let offset = 0; offset < band; offset++) {
      ring.push(alphaAt(data, width, offset, y), alphaAt(data, width, width - 1 - offset, y));
    }
  }

  if (share(ring, (alpha) => alpha >= OPAQUE) >= 0.9) return "filled";

  const corner = Math.max(1, Math.round(Math.min(width, height) / 10));
  const corners: number[] = [];

  for (let dy = 0; dy < corner; dy++) {
    for (let dx = 0; dx < corner; dx++) {
      corners.push(
        alphaAt(data, width, dx, dy),
        alphaAt(data, width, width - 1 - dx, dy),
        alphaAt(data, width, dx, height - 1 - dy),
        alphaAt(data, width, width - 1 - dx, height - 1 - dy),
      );
    }
  }

  const midX = Math.floor(width / 2);
  const midY = Math.floor(height / 2);
  const middles = [
    alphaAt(data, width, midX, band),
    alphaAt(data, width, midX, height - 1 - band),
    alphaAt(data, width, band, midY),
    alphaAt(data, width, width - 1 - band, midY),
  ];

  if (
    share(corners, (alpha) => alpha <= CLEAR) >= 0.8 &&
    share(middles, (alpha) => alpha >= OPAQUE) >= 0.75
  ) {
    return "badge";
  }

  return "symbol";
}

/**
 * The logo's brand color as hex, or null when the mark is monochrome (black, white, grey).
 * Opaque, saturated pixels vote into twelve hue buckets weighted by saturation; the
 * heaviest bucket's average wins if it covers enough of the mark to be its identity.
 */
export function dominantColor(
  data: ArrayLike<number>,
  width: number,
  height: number,
): string | null {
  const buckets = Array.from({ length: 12 }, () => ({ weight: 0, r: 0, g: 0, b: 0 }));
  let opaque = 0;

  for (let index = 0; index < width * height; index++) {
    const offset = index * 4;
    if (data[offset + 3] < OPAQUE) continue;

    opaque++;

    const r = data[offset] / 255;
    const g = data[offset + 1] / 255;
    const b = data[offset + 2] / 255;
    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    const saturation = max === 0 ? 0 : (max - min) / max;

    if (saturation < 0.35 || max < 0.2) continue;

    const delta = max - min;
    const hue =
      max === r ? ((g - b) / delta + 6) % 6 : max === g ? (b - r) / delta + 2 : (r - g) / delta + 4;
    const bucket = buckets[Math.floor(hue * 2) % 12];

    bucket.weight += saturation;
    bucket.r += r * saturation;
    bucket.g += g * saturation;
    bucket.b += b * saturation;
  }

  const best = buckets.reduce((top, bucket) => (bucket.weight > top.weight ? bucket : top));
  if (opaque === 0 || best.weight / opaque < 0.08) return null;

  const channel = (value: number) =>
    Math.round((value / best.weight) * 255)
      .toString(16)
      .padStart(2, "0");

  return `#${channel(best.r)}${channel(best.g)}${channel(best.b)}`;
}
