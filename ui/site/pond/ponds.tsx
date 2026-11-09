import { type RefObject, useEffect, useRef } from "react";
import { type FishSpec, PondEngine, type Slot } from "./engine";

type Zone = { x: number; y: number; rx: number; ry: number };

const SHOAL_ASPECT = 0.62;

/** The panel's box in the pond's coordinates, kept clear of fish. */
function zoneOf(panel: HTMLElement | null, canvas: HTMLCanvasElement | null): Zone | null {
  const box = panel?.getBoundingClientRect();
  const origin = canvas?.getBoundingClientRect();

  if (!box || !origin || box.width === 0) {
    return null;
  }

  return {
    x: box.left - origin.left + box.width / 2,
    y: box.top - origin.top + box.height / 2,
    rx: box.width / 2,
    ry: box.height / 2,
  };
}

/** Stable, scattered orange: about one fish in five, without stripes across the rows. */
const isNamed = (index: number) => {
  const value = Math.sin(index * 12.9898) * 43758.5453;

  return value - Math.floor(value) < 0.2;
};

/**
 * Every slot of a shoal at one spacing, in staggered rows. Beside the panel only the wide side holds
 * fish: a thin column on the other reads as a mistake.
 */
function slotsAt(pond: PondEngine, zone: Zone | null, spacing: number) {
  const rowHeight = spacing * SHOAL_ASPECT;
  const slots: Slot[] = [];

  for (let row = 0; row * rowHeight + rowHeight / 2 < pond.height; row++) {
    const y = row * rowHeight + rowHeight / 2;

    for (let column = 0; column * spacing < pond.width; column++) {
      const x = column * spacing + (row % 2 ? spacing * 0.5 : 0) + spacing * 0.42;
      const beside = zone !== null && Math.abs(y - zone.y) <= zone.ry + rowHeight * 0.5;
      const right = (zone?.x ?? 0) + (zone?.rx ?? 0) + spacing * 0.65;
      const clear = !beside || x - spacing * 0.2 > right;

      if (clear && x + spacing * 0.5 < pond.width) {
        slots.push({ x, y });
      }
    }
  }

  return slots;
}

/** The largest spacing that seats the whole shoal, the slots nearest the middle, and a dot pitch. */
function shoal(count: number, pond: PondEngine, zone: Zone | null) {
  let spacing = 190;

  while (spacing > 14 && slotsAt(pond, zone, spacing).length < count) {
    spacing -= 2;
  }

  const size = spacing * 0.68;
  const middle = { x: pond.width * 0.62, y: pond.height / 2 };
  const slots = slotsAt(pond, zone, spacing)
    .sort(
      (left, right) =>
        Math.hypot(left.x - middle.x, left.y - middle.y) -
        Math.hypot(right.x - middle.x, right.y - middle.y),
    )
    .slice(0, count);
  const specs: FishSpec[] = slots.map((_, index) => ({
    named: isNamed(index),
    depth: 1 + (index % 2),
    size,
  }));

  return { specs, slots, step: Math.max(2, Math.min(5, Math.round((size / 15) * 2) / 2)) };
}

/**
 * A pond beside a panel. In pricing it holds one koi for each prompt you pay for, and the slider adds
 * and removes them; behind the closing call to action it holds a fixed shoal.
 */
export function Pond({ count, avoid }: { count: number; avoid: RefObject<HTMLElement | null> }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const pond = useRef<PondEngine | null>(null);
  const amount = useRef(count);

  useEffect(() => {
    if (!canvas.current) {
      return;
    }

    const form = (engine: PondEngine) => {
      const { specs, slots, step } = shoal(
        amount.current,
        engine,
        zoneOf(avoid.current, canvas.current),
      );

      engine.setStep(step);
      engine.setShoal(specs, slots);
    };
    const engine = new PondEngine(canvas.current, { step: 5, onResize: form });

    pond.current = engine;
    form(engine);

    return () => {
      engine.destroy();
      pond.current = null;
    };
  }, [avoid]);

  useEffect(() => {
    amount.current = count;

    const engine = pond.current;

    if (!engine) {
      return;
    }

    const { specs, slots, step } = shoal(count, engine, zoneOf(avoid.current, canvas.current));

    engine.setStep(step);
    engine.setShoal(specs, slots);
  }, [count, avoid]);

  return <canvas ref={canvas} className="absolute inset-0 size-full" />;
}
