/**
 * The pond: koi drawn as a halftone on black. Each fish is the mark's anatomy (a plump body, a
 * narrow waist, two round tail lobes, two round fins, one eye). The fish are painted into a tiny
 * canvas with one pixel per dot, so a pixel's coverage becomes its dot's size. That keeps the
 * silhouette readable at a coarse grid, where sampling a yes or no at each point would break it up.
 *
 * The fish form a shoal: each holds a slot in staggered rows and faces the same way, so they never
 * pile up or overlap, and hundreds still read as fish. A slow swell runs through the rows.
 */

const SPINE = 16;
const ORANGE = "#ff5c12";
const WATER = "#17171a";
const GREYS = [0.24, 0.4, 0.62];
const GREY_LEVELS = 8;
const FRAME_MS = 30;

export type FishSpec = {
  named: boolean;
  size: number;
  depth: number;
};

export type Slot = { x: number; y: number };
type Ring = { x: number; y: number; radius: number; age: number };

export type PondOptions = {
  step: number;
  onResize?: (pond: PondEngine) => void;
};

const easeOut = (value: number) => 1 - (1 - value) ** 3;

export class Fish {
  named: boolean;
  depth: number;
  size: number;
  targetSize: number;
  x: number;
  y: number;
  angle = 0;
  phase: number;
  life = 1;
  leaving = false;
  slot: Slot;
  points: { x: number; y: number }[];

  constructor(spec: FishSpec, slot: Slot, phase: number) {
    this.named = spec.named;
    this.depth = spec.depth;
    this.size = spec.size;
    this.targetSize = spec.size;
    this.slot = slot;
    this.x = slot.x;
    this.y = slot.y;
    this.phase = phase;
    this.points = Array.from({ length: SPINE }, (_, index) => ({
      x: slot.x - (index * spec.size) / SPINE,
      y: slot.y,
    }));
  }

  /** Lays the body out straight behind the head, as when a fish is placed on its start line. */
  straighten() {
    for (let index = 0; index < SPINE; index++) {
      this.points[index].x = this.x - (Math.cos(this.angle) * index * this.size) / SPINE;
      this.points[index].y = this.y - (Math.sin(this.angle) * index * this.size) / SPINE;
    }
  }

  /** Paints the body, fins, tail and eye. Every part winds the same way, so overlaps stay solid. */
  paint(context: CanvasRenderingContext2D, fill: string) {
    const size = this.size;
    const spine = this.points.map((point, index) => {
      const t = index / (SPINE - 1);
      const next = this.points[Math.min(index + 1, SPINE - 1)];
      const previous = this.points[Math.max(index - 1, 0)];
      const dx = next.x - previous.x;
      const dy = next.y - previous.y;
      const length = Math.hypot(dx, dy) || 1;
      const nx = -dy / length;
      const ny = dx / length;
      const sway = Math.sin(this.phase - t * 3.6) * size * 0.11 * t ** 1.4;

      return {
        x: point.x + nx * sway,
        y: point.y + ny * sway,
        nx,
        ny,
        tx: -dx / length,
        ty: -dy / length,
      };
    });
    const girth = size * 0.215;
    const width = (t: number) => {
      const head = 0.34;

      if (t < head) {
        const u = (head - t) / head;

        return girth * Math.sqrt(Math.max(0, 1 - u * u));
      }

      const u = (t - head) / (1 - head);

      return girth * (0.2 + 0.8 * Math.cos((u * Math.PI) / 2) ** 1.5);
    };
    const left = spine.map((point, index) => {
      const offset = width(index / (SPINE - 1));

      return [point.x + point.nx * offset, point.y + point.ny * offset];
    });
    const right = spine.map((point, index) => {
      const offset = width(index / (SPINE - 1));

      return [point.x - point.nx * offset, point.y - point.ny * offset];
    });
    const outline = [...left, ...right.reverse()];

    context.beginPath();
    context.moveTo(
      (outline[0][0] + outline[outline.length - 1][0]) / 2,
      (outline[0][1] + outline[outline.length - 1][1]) / 2,
    );

    for (let index = 0; index < outline.length; index++) {
      const a = outline[index];
      const b = outline[(index + 1) % outline.length];

      context.quadraticCurveTo(a[0], a[1], (a[0] + b[0]) / 2, (a[1] + b[1]) / 2);
    }

    context.closePath();

    const tail = spine[SPINE - 1];
    const backward = Math.atan2(-tail.ty, -tail.tx);
    const flick = Math.sin(this.phase - 3.6) * 0.22;

    for (const side of [-1, 1]) {
      const angle = backward + side * 0.58 + flick;
      const length = size * 0.52;
      const x = tail.x + Math.cos(angle) * length * 0.46;
      const y = tail.y + Math.sin(angle) * length * 0.46;

      context.moveTo(x + (Math.cos(angle) * length) / 2, y + (Math.sin(angle) * length) / 2);
      context.ellipse(x, y, length / 2, size * 0.105, angle, 0, Math.PI * 2, true);
    }

    const shoulder = spine[Math.round(SPINE * 0.3)];
    const forward = Math.atan2(shoulder.ty, shoulder.tx);
    const beat = Math.sin(this.phase * 0.7) * 0.16;

    for (const side of [-1, 1]) {
      const angle = forward + Math.PI + side * (0.82 + beat);
      const length = size * 0.22;
      const offset = width(0.3) * 0.82;
      const x = shoulder.x + shoulder.nx * side * offset + Math.cos(angle) * length * 0.42;
      const y = shoulder.y + shoulder.ny * side * offset + Math.sin(angle) * length * 0.42;

      context.moveTo(x + (Math.cos(angle) * length) / 2, y + (Math.sin(angle) * length) / 2);
      context.ellipse(x, y, length / 2, size * 0.062, angle, 0, Math.PI * 2, true);
    }

    const crown = spine[3];
    const radius = width(3 / (SPINE - 1)) * 1.02;

    context.moveTo(crown.x + radius, crown.y);
    context.arc(crown.x, crown.y, radius, 0, Math.PI * 2, true);
    context.fillStyle = fill;
    context.fill("nonzero");

    return spine[2];
  }

  eye(
    context: CanvasRenderingContext2D,
    at: { x: number; y: number; nx: number; ny: number },
    step: number,
  ) {
    context.beginPath();
    context.arc(
      at.x + at.nx * this.size * 0.085,
      at.y + at.ny * this.size * 0.085,
      Math.max(step * 0.62, this.size * 0.042),
      0,
      Math.PI * 2,
    );
    context.fillStyle = "#000";
    context.fill();
  }
}

export class PondEngine {
  fish: Fish[] = [];
  width = 0;
  height = 0;
  private canvas: HTMLCanvasElement;
  private context: CanvasRenderingContext2D;
  private options: PondOptions;
  private step: number;
  private cells = document.createElement("canvas");
  private cellContext: CanvasRenderingContext2D;
  private water = document.createElement("canvas");
  private rings: Ring[] = [];
  private ratio = 1;
  private frame = 0;
  private last = 0;
  private clock = 0;
  private running = false;
  private observer: IntersectionObserver;
  private resizer: ResizeObserver;
  readonly still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  constructor(canvas: HTMLCanvasElement, options: PondOptions) {
    this.canvas = canvas;
    this.options = options;
    this.step = options.step;
    this.context = canvas.getContext("2d") as CanvasRenderingContext2D;
    this.cellContext = this.cells.getContext("2d", {
      willReadFrequently: true,
    }) as CanvasRenderingContext2D;
    this.measure();
    this.observer = new IntersectionObserver(([entry]) => {
      entry.isIntersecting ? this.start() : this.stop();
    });
    this.observer.observe(canvas);
    this.resizer = new ResizeObserver(() => {
      const box = this.canvas.getBoundingClientRect();

      if (Math.abs(box.width - this.width) < 1 && Math.abs(box.height - this.height) < 1) {
        return;
      }

      this.measure();
      this.options.onResize?.(this);
      this.draw();
    });
    this.resizer.observe(canvas);
  }

  destroy() {
    this.stop();
    this.observer.disconnect();
    this.resizer.disconnect();
  }

  private measure() {
    const box = this.canvas.getBoundingClientRect();

    this.width = box.width;
    this.height = box.height;
    this.ratio = Math.min(window.devicePixelRatio || 1, 2);
    this.canvas.width = Math.round(box.width * this.ratio);
    this.canvas.height = Math.round(box.height * this.ratio);
    this.paintWater();
  }

  private paintWater() {
    const { step, width, height, ratio, water } = this;

    water.width = Math.max(1, Math.round(width * ratio));
    water.height = Math.max(1, Math.round(height * ratio));

    const context = water.getContext("2d") as CanvasRenderingContext2D;
    const radius = Math.max(0.42, step * 0.1);

    context.scale(ratio, ratio);
    context.fillStyle = WATER;
    context.beginPath();

    for (let y = step / 2; y < height; y += step) {
      for (let x = step / 2; x < width; x += step) {
        context.moveTo(x + radius, y);
        context.arc(x, y, radius, 0, Math.PI * 2);
      }
    }

    context.fill();
    this.cells.width = Math.ceil(width / step) + 1;
    this.cells.height = Math.ceil(height / step) + 1;
  }

  setStep(step: number) {
    if (step !== this.step) {
      this.step = step;
      this.paintWater();
    }
  }

  /**
   * Seats the shoal. Fish already in the pond swim to their new slot and size; new ones fade in
   * with a ripple, and the ones no longer needed fade out where they are.
   */
  setShoal(specs: FishSpec[], slots: Slot[]) {
    const kept = this.fish.filter((fish) => !fish.leaving);
    const growing = kept.length > 0 && specs.length - kept.length <= 30;

    for (let index = specs.length; index < kept.length; index++) {
      kept[index].leaving = true;
    }

    for (let index = 0; index < specs.length; index++) {
      const slot = slots[index];
      const fish = kept[index];

      if (fish) {
        fish.slot = slot;
        fish.targetSize = specs[index].size;
        continue;
      }

      const born = new Fish(specs[index], slot, index * 0.37);

      born.life = kept.length > 0 ? 0 : 1;
      this.fish.push(born);

      if (growing) {
        this.ripple(slot.x, slot.y, specs[index].size * 1.1);
      }
    }

    if (this.still) {
      this.fish = this.fish.filter((fish) => !fish.leaving);

      for (const fish of this.fish) {
        fish.life = 1;
        fish.size = fish.targetSize;
        fish.x = fish.slot.x;
        fish.y = fish.slot.y;
        fish.straighten();
      }
    }

    this.draw();
  }

  private ripple(x: number, y: number, radius: number) {
    if (this.still) {
      return;
    }

    this.rings.push({ x, y, radius, age: 0 });

    if (this.rings.length > 14) {
      this.rings.shift();
    }
  }

  private start() {
    if (this.running) {
      return;
    }

    this.running = true;
    this.last = performance.now();
    this.frame = requestAnimationFrame(this.loop);
  }

  private stop() {
    this.running = false;
    cancelAnimationFrame(this.frame);
  }

  private loop = (now: number) => {
    if (!this.running) {
      return;
    }

    this.frame = requestAnimationFrame(this.loop);

    const elapsed = now - this.last;

    if (elapsed < FRAME_MS) {
      return;
    }

    this.last = now;

    if (!this.still) {
      this.tick(Math.min(3, elapsed / 16.67));
    }

    this.draw();
  };

  private tick(delta: number) {
    this.clock += delta;

    for (const fish of this.fish) {
      fish.size += (fish.targetSize - fish.size) * Math.min(1, 0.06 * delta);

      if (fish.leaving) {
        fish.life -= 0.04 * delta;
      } else if (fish.life < 1) {
        fish.life = Math.min(1, fish.life + 0.03 * delta);
      }

      // A slow swell passes through the rows, so the shoal reads as one living body of fish.
      const wave = this.clock * 0.022 + fish.slot.x * 0.006;
      const targetX = fish.slot.x + Math.cos(wave) * 3;
      const targetY = fish.slot.y + Math.sin(wave + fish.slot.y * 0.01) * 5;

      fish.x += (targetX - fish.x) * Math.min(1, 0.11 * delta);
      fish.y += (targetY - fish.y) * Math.min(1, 0.11 * delta);
      fish.phase += 0.07 * delta;

      const segment = fish.size / SPINE;
      const settle = Math.min(1, 0.05 * delta);

      fish.points[0].x = fish.x;
      fish.points[0].y = fish.y;

      for (let index = 1; index < SPINE; index++) {
        const point = fish.points[index];
        const ahead = fish.points[index - 1];
        const dx = point.x - ahead.x;
        const dy = point.y - ahead.y;
        const length = Math.hypot(dx, dy) || 1;

        point.x = ahead.x + (dx / length) * segment;
        point.y = ahead.y + (dy / length) * segment;

        // The body settles straight behind the head, so every fish ends up facing forward.
        point.x += (fish.x - index * segment - point.x) * settle;
        point.y += (fish.y - point.y) * settle;
      }
    }

    this.fish = this.fish.filter((fish) => fish.life > 0);

    for (const ring of this.rings) {
      ring.age = Math.min(1, ring.age + delta / 100);
    }

    this.rings = this.rings.filter((ring) => ring.age < 1);
  }

  draw() {
    const context = this.context;
    const cell = this.cellContext;
    const { step, width, height, ratio } = this;

    if (width === 0 || height === 0) {
      return;
    }

    context.setTransform(ratio, 0, 0, ratio, 0, 0);
    context.clearRect(0, 0, width, height);
    context.drawImage(this.water, 0, 0, width, height);

    cell.setTransform(1 / step, 0, 0, 1 / step, 0, 0);
    cell.globalAlpha = 1;
    cell.fillStyle = "#000";
    cell.fillRect(0, 0, width + step * 2, height + step * 2);

    for (const ring of this.rings) {
      cell.beginPath();
      cell.arc(ring.x, ring.y, ring.radius * easeOut(ring.age), 0, Math.PI * 2);
      cell.strokeStyle = `rgba(255,255,255,${0.34 * (1 - ring.age)})`;
      cell.lineWidth = step * 1.3;
      cell.stroke();
    }

    const order = [...this.fish].sort(
      (left, right) => (left.named ? 3 : left.depth) - (right.named ? 3 : right.depth),
    );

    for (const fish of order) {
      const grey = Math.round(GREYS[fish.depth] * 255);

      cell.globalAlpha = Math.max(0, Math.min(1, fish.life));

      const eye = fish.paint(cell, fish.named ? ORANGE : `rgb(${grey},${grey},${grey})`);

      fish.eye(cell, eye, step);
    }

    const columns = this.cells.width;
    const rows = this.cells.height;
    const data = cell.getImageData(0, 0, columns, rows).data;
    const maximum = step * 0.47;
    const greys = Array.from({ length: GREY_LEVELS }, () => [] as number[]);
    const oranges: number[] = [];

    for (let row = 0; row < rows; row++) {
      for (let column = 0; column < columns; column++) {
        const index = (row * columns + column) * 4;
        const red = data[index];
        const green = data[index + 1];

        if (red < 8) {
          continue;
        }

        const x = (column + 0.5) * step;
        const y = (row + 0.5) * step;

        if (red > green * 1.5 && red > 28) {
          oranges.push(x, y, maximum * Math.min(1, 0.25 + 0.85 * Math.sqrt(red / 255)));
          continue;
        }

        const level = red / 255;
        const bucket = Math.min(GREY_LEVELS - 1, Math.floor(level * GREY_LEVELS));

        greys[bucket].push(x, y, maximum * Math.min(1, 0.22 + 0.9 * Math.sqrt(level / GREYS[2])));
      }
    }

    greys.forEach((dots, bucket) => {
      if (dots.length > 0) {
        const value = Math.round(((bucket + 1) / GREY_LEVELS) * 240);

        this.dots(dots, `rgb(${value},${value},${value + 4})`);
      }
    });

    this.dots(oranges, ORANGE);
  }

  private dots(dots: number[], fill: string) {
    const context = this.context;

    context.fillStyle = fill;
    context.beginPath();

    for (let index = 0; index < dots.length; index += 3) {
      context.moveTo(dots[index] + dots[index + 2], dots[index + 1]);
      context.arc(dots[index], dots[index + 1], dots[index + 2], 0, Math.PI * 2);
    }

    context.fill();
  }
}
