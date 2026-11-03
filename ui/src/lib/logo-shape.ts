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
