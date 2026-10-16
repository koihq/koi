import { describe, expect, it } from "vitest";
import { smoothSegments, trendAreaPath, trendPath } from "./trend-path";

describe("visibility trend geometry", () => {
  it("passes through observations and never overshoots consecutive values", () => {
    const values = [0, 100, 15, 15, 16, 95, 96, 0, 0];
    const points = values.map((y, i) => ({ x: i * 30, y }));
    const curves = smoothSegments(points);

    expect(curves).toHaveLength(points.length - 1);
    for (const [index, curve] of curves.entries()) {
      expect(curve.from).toEqual(points[index]);
      expect(curve.to).toEqual(points[index + 1]);
      for (let step = 0; step <= 100; step++) {
        const t = step / 100;
        const y =
          (1 - t) ** 3 * curve.from.y +
          3 * (1 - t) ** 2 * t * curve.first.y +
          3 * (1 - t) * t ** 2 * curve.second.y +
          t ** 3 * curve.to.y;
        expect(y).toBeGreaterThanOrEqual(Math.min(curve.from.y, curve.to.y) - 1e-10);
        expect(y).toBeLessThanOrEqual(Math.max(curve.from.y, curve.to.y) + 1e-10);
      }
    }
  });

  it("starts a new path after gaps and retains isolated zero observations", () => {
    const path = trendPath([
      { x: 0, y: 10 },
      { x: 20, y: 20 },
      null,
      { x: 60, y: 0 },
      null,
      { x: 100, y: 50 },
      { x: 120, y: 60 },
    ]);

    expect(path.match(/M/g)).toHaveLength(3);
    expect(path.match(/C/g)).toHaveLength(2);
    expect(path).toContain("M60.00,0.00");
    expect(trendPath([null, null])).toBe("");
    expect(smoothSegments([{ x: 0, y: 0 }])).toEqual([]);
  });

  it("fills only observed runs and closes each one at the baseline", () => {
    const area = trendAreaPath(
      [
        { x: 0, y: 10 },
        { x: 20, y: 20 },
        null,
        { x: 60, y: 30 },
        null,
        { x: 100, y: 40 },
        { x: 120, y: 50 },
      ],
      100,
    );

    expect(area.match(/M/g)).toHaveLength(2);
    expect(area.match(/Z/g)).toHaveLength(2);
    expect(area).toContain("L20.00,100.00 L0.00,100.00 Z");
    expect(area).toContain("L120.00,100.00 L100.00,100.00 Z");
    expect(area).not.toContain("M60.00,30.00");
    expect(trendAreaPath([null, { x: 60, y: 30 }], 100)).toBe("");
  });
});
