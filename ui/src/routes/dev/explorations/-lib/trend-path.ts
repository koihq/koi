export type PlotPoint = { x: number; y: number };
export type CurveSegment = {
  from: PlotPoint;
  to: PlotPoint;
  first: PlotPoint;
  second: PlotPoint;
};

// Monotone Hermite interpolation: retain the observations without inventing extrema.
export function smoothSegments(points: PlotPoint[]): CurveSegment[] {
  if (points.length < 2) return [];

  const slopes = points.slice(1).map((point, index) => {
    const previous = points[index];
    return (point.y - previous.y) / (point.x - previous.x);
  });
  const tangents = points.map((_, index) => {
    if (index === 0) return slopes[0];
    if (index === points.length - 1) return slopes[index - 1];

    const before = slopes[index - 1];
    const after = slopes[index];
    return before * after <= 0 ? 0 : (2 * before * after) / (before + after);
  });

  for (let index = 0; index < slopes.length; index++) {
    const slope = slopes[index];
    if (slope === 0) {
      tangents[index] = 0;
      tangents[index + 1] = 0;
      continue;
    }

    const length = Math.hypot(tangents[index] / slope, tangents[index + 1] / slope);
    if (length > 3) {
      tangents[index] *= 3 / length;
      tangents[index + 1] *= 3 / length;
    }
  }

  return points.slice(1).map((to, index) => {
    const from = points[index];
    const third = (to.x - from.x) / 3;
    return {
      from,
      to,
      first: { x: from.x + third, y: from.y + third * tangents[index] },
      second: { x: to.x - third, y: to.y - third * tangents[index + 1] },
    };
  });
}

export function trendPath(points: (PlotPoint | null)[]) {
  const runs: PlotPoint[][] = [];
  let run: PlotPoint[] = [];

  for (const point of points) {
    if (point === null) {
      run = [];
    } else {
      if (!run.length) runs.push(run);
      run.push(point);
    }
  }

  const pair = (point: PlotPoint) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`;
  return runs
    .map((points) => {
      const curves = smoothSegments(points).map(
        ({ first, second, to }) => `C${pair(first)} ${pair(second)} ${pair(to)}`,
      );
      return `M${pair(points[0])} ${curves.join(" ")}`;
    })
    .join(" ");
}
