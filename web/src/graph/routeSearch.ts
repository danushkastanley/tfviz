import type { Point } from "./edgePath";
import type { RouteGrid } from "./routeGrid";

/** Right, down, left, up. */
export type Direction = 0 | 1 | 2 | 3;
const DI = [1, 0, -1, 0] as const;
const DJ = [0, 1, 0, -1] as const;
const DIRECTIONS: readonly Direction[] = [0, 1, 2, 3];

/** Each bend costs as much as this many pixels of line, so routes stay simple. */
const BEND_COST = 40;

/** A lane crossing just outside a card side, and the direction away from the card. */
export interface Port {
  x: number;
  y: number;
  outward: Direction;
}

const opposite = (d: Direction): Direction => ((d + 2) % 4) as Direction;

/** Binary min-heap on (cost, insertion order), so equal costs resolve deterministically. */
class Queue {
  private readonly items: { cost: number; seq: number; state: number }[] = [];
  private seq = 0;

  get size(): number {
    return this.items.length;
  }

  push(cost: number, state: number): void {
    const items = this.items;
    items.push({ cost, seq: this.seq++, state });
    let i = items.length - 1;
    while (i > 0) {
      const parent = (i - 1) >> 1;
      if (!this.less(i, parent)) break;
      this.swap(i, parent);
      i = parent;
    }
  }

  pop(): number {
    const items = this.items;
    const top = items[0] as { state: number };
    const last = items.pop() as (typeof items)[number];
    if (items.length > 0) {
      items[0] = last;
      let i = 0;
      for (;;) {
        const l = 2 * i + 1;
        const r = l + 1;
        let best = i;
        if (l < items.length && this.less(l, best)) best = l;
        if (r < items.length && this.less(r, best)) best = r;
        if (best === i) break;
        this.swap(i, best);
        i = best;
      }
    }
    return top.state;
  }

  private less(a: number, b: number): boolean {
    const x = this.items[a] as { cost: number; seq: number };
    const y = this.items[b] as { cost: number; seq: number };
    return x.cost < y.cost || (x.cost === y.cost && x.seq < y.seq);
  }

  private swap(a: number, b: number): void {
    const items = this.items;
    [items[a], items[b]] = [items[b] as (typeof items)[number], items[a] as (typeof items)[number]];
  }
}

/**
 * A* over the grid's lane crossings. A state is a crossing plus the direction
 * of travel, so bends can be priced. The route leaves through one of the
 * source ports and arrives head-on through one of the target ports.
 */
export function searchRoute(grid: RouteGrid, sources: readonly Port[], targets: readonly Port[]): Point[] | undefined {
  const nx = grid.xs.length;
  const nodes = nx * grid.ys.length;
  const cost = new Float64Array(nodes * 4).fill(Infinity);
  const previous = new Int32Array(nodes * 4).fill(-1);
  const arrival = new Map<number, Direction>();
  for (const t of targets) {
    const i = grid.indexOfX(t.x);
    const j = grid.indexOfY(t.y);
    if (i >= 0 && j >= 0 && !grid.isBlocked(i, j)) arrival.set(j * nx + i, opposite(t.outward));
  }
  if (arrival.size === 0) return undefined;

  const goals = targets.map((t) => ({ x: t.x, y: t.y }));
  const estimate = (node: number) => {
    const x = grid.xs[node % nx] as number;
    const y = grid.ys[Math.floor(node / nx)] as number;
    return Math.min(...goals.map((g) => Math.abs(g.x - x) + Math.abs(g.y - y)));
  };

  const queue = new Queue();
  const relax = (state: number, value: number, from: number) => {
    if (value >= (cost[state] as number)) return;
    cost[state] = value;
    previous[state] = from;
    queue.push(value + estimate(state >> 2), state);
  };
  for (const s of sources) {
    const i = grid.indexOfX(s.x);
    const j = grid.indexOfY(s.y);
    if (i >= 0 && j >= 0 && !grid.isBlocked(i, j)) relax((j * nx + i) * 4 + s.outward, 0, -1);
  }

  const settled = new Uint8Array(nodes * 4);
  while (queue.size > 0) {
    const state = queue.pop();
    if (settled[state]) continue;
    settled[state] = 1;
    const node = state >> 2;
    const heading = (state & 3) as Direction;
    const g = cost[state] as number;
    const inward = arrival.get(node);
    if (inward === heading) return trace(grid, previous, state);
    if (inward !== undefined) relax(node * 4 + inward, g + BEND_COST, state);

    const i = node % nx;
    const j = Math.floor(node / nx);
    for (const d of DIRECTIONS) {
      if (d === opposite(heading)) continue;
      const ni = i + DI[d];
      const nj = j + DJ[d];
      if (ni < 0 || nj < 0 || ni >= nx || nj >= grid.ys.length) continue;
      if (grid.isStepBlocked(i, j, DI[d], DJ[d]) || grid.isBlocked(ni, nj)) continue;
      const length = Math.abs((grid.xs[ni] as number) - (grid.xs[i] as number)) + Math.abs((grid.ys[nj] as number) - (grid.ys[j] as number));
      relax((nj * nx + ni) * 4 + d, g + length + (d === heading ? 0 : BEND_COST), state);
    }
  }
  return undefined;
}

function trace(grid: RouteGrid, previous: Int32Array, end: number): Point[] {
  const nx = grid.xs.length;
  const points: Point[] = [];
  for (let state = end; state >= 0; state = previous[state] as number) {
    const node = state >> 2;
    points.push({ x: grid.xs[node % nx] as number, y: grid.ys[Math.floor(node / nx)] as number });
  }
  return points.reverse();
}
