import type { Rect } from "./edgePath";

/**
 * Lanes keep this distance from every card. It is half the layout's gap
 * between cards, so a lane runs down the middle of each gutter.
 */
export const CLEARANCE = 6;

const EPSILON = 0.01;

/**
 * A sparse orthogonal grid: lanes beside every obstacle plus any extra
 * coordinates the caller needs (ports). Movement is only between
 * neighbouring lane crossings, so a route can never cut through a card.
 */
export interface RouteGrid {
  xs: readonly number[];
  ys: readonly number[];
  /** True when the crossing (i, j) lies inside an obstacle. */
  isBlocked(i: number, j: number): boolean;
  /** True when the step from (i, j) to the neighbouring crossing crosses an obstacle. */
  isStepBlocked(i: number, j: number, di: number, dj: number): boolean;
  indexOfX(x: number): number;
  indexOfY(y: number): number;
}

function lanes(values: number[]): number[] {
  values.sort((a, b) => a - b);
  const out: number[] = [];
  for (const v of values) {
    const last = out[out.length - 1];
    if (last === undefined || v - last > EPSILON) out.push(v);
  }
  return out;
}

/** Lane values with the midpoint between each neighbouring pair interleaved. */
function doubled(values: readonly number[]): number[] {
  const out: number[] = [];
  values.forEach((v, i) => {
    if (i > 0) out.push(((values[i - 1] as number) + v) / 2);
    out.push(v);
  });
  return out;
}

/** First index whose value is greater than `limit`. */
function firstAbove(values: readonly number[], limit: number): number {
  let lo = 0;
  let hi = values.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if ((values[mid] as number) > limit) hi = mid;
    else lo = mid + 1;
  }
  return lo;
}

function indexOf(values: readonly number[], v: number): number {
  const i = firstAbove(values, v - EPSILON);
  return i < values.length && Math.abs((values[i] as number) - v) <= EPSILON ? i : -1;
}

export function buildGrid(obstacles: readonly Rect[], bounds: Rect, extraXs: readonly number[], extraYs: readonly number[]): RouteGrid {
  const xValues = [bounds.x, bounds.x + bounds.width, ...extraXs];
  const yValues = [bounds.y, bounds.y + bounds.height, ...extraYs];
  for (const o of obstacles) {
    xValues.push(o.x - CLEARANCE, o.x + o.width + CLEARANCE);
    yValues.push(o.y - CLEARANCE, o.y + o.height + CLEARANCE);
  }
  const xs = lanes(xValues);
  const ys = lanes(yValues);

  // Crossings sit at even indices and the steps between them at odd ones,
  // so one bitmap answers both "is this point free" and "is this step free".
  const dx = doubled(xs);
  const dy = doubled(ys);
  const width = dx.length;
  const blocked = new Uint8Array(width * dy.length);
  for (const o of obstacles) {
    // Lanes lie exactly on the inflated boundary and stay open.
    const x0 = firstAbove(dx, o.x - CLEARANCE + EPSILON);
    const x1 = firstAbove(dx, o.x + o.width + CLEARANCE - EPSILON);
    const y0 = firstAbove(dy, o.y - CLEARANCE + EPSILON);
    const y1 = firstAbove(dy, o.y + o.height + CLEARANCE - EPSILON);
    for (let j = y0; j < y1; j++) blocked.fill(1, j * width + x0, j * width + x1);
  }

  return {
    xs,
    ys,
    isBlocked: (i, j) => blocked[2 * j * width + 2 * i] === 1,
    isStepBlocked: (i, j, di, dj) => blocked[(2 * j + dj) * width + 2 * i + di] === 1,
    indexOfX: (x) => indexOf(xs, x),
    indexOfY: (y) => indexOf(ys, y),
  };
}
