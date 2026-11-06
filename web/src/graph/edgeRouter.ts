import { contains, curvePath, roundedPath, type Point, type Rect } from "./edgePath";
import { buildGrid, CLEARANCE } from "./routeGrid";
import { searchRoute, type Port } from "./routeSearch";

/** How far beyond the two endpoints a route may first look for a way round. */
const REACH = 160;

interface Side {
  /** Where the line meets the card. */
  edge: Point;
  port: Port;
}

/** The midpoint of each side, and the lane crossing just outside it. */
function sides(r: Rect): Side[] {
  const cx = r.x + r.width / 2;
  const cy = r.y + r.height / 2;
  const right = r.x + r.width;
  const bottom = r.y + r.height;
  return [
    { edge: { x: right, y: cy }, port: { x: right + CLEARANCE, y: cy, outward: 0 } },
    { edge: { x: cx, y: bottom }, port: { x: cx, y: bottom + CLEARANCE, outward: 1 } },
    { edge: { x: r.x, y: cy }, port: { x: r.x - CLEARANCE, y: cy, outward: 2 } },
    { edge: { x: cx, y: r.y }, port: { x: cx, y: r.y - CLEARANCE, outward: 3 } },
  ];
}

function union(rects: readonly Rect[], margin: number): Rect {
  const x = Math.min(...rects.map((r) => r.x)) - margin;
  const y = Math.min(...rects.map((r) => r.y)) - margin;
  const right = Math.max(...rects.map((r) => r.x + r.width)) + margin;
  const bottom = Math.max(...rects.map((r) => r.y + r.height)) + margin;
  return { x, y, width: right - x, height: bottom - y };
}

function overlaps(a: Rect, b: Rect): boolean {
  return a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height;
}

function routeWithin(obstacles: readonly Rect[], bounds: Rect, source: Rect, target: Rect): Point[] | undefined {
  const near = obstacles.filter((o) => overlaps(o, bounds) && !overlaps(o, source) && !overlaps(o, target));
  const from = sides(source);
  const to = sides(target);
  const ports = [...from, ...to].map((s) => s.port);
  // The endpoints are obstacles too, so a line never doubles back across its own card.
  const grid = buildGrid([...near, source, target], bounds, ports.map((p) => p.x), ports.map((p) => p.y));
  const lane = searchRoute(grid, from.map((s) => s.port), to.map((s) => s.port));
  const start = lane?.[0];
  const end = lane?.[lane.length - 1];
  if (!lane || !start || !end) return undefined;
  const exit = from.find((s) => s.port.x === start.x && s.port.y === start.y) as Side;
  const entry = to.find((s) => s.port.x === end.x && s.port.y === end.y) as Side;
  return [exit.edge, ...lane, entry.edge];
}

/**
 * The points of an orthogonal route between two rectangles that keeps clear
 * of every obstacle, or undefined when there is none. It looks near the
 * endpoints first and then across the whole canvas.
 */
export function routeOrthogonal(obstacles: readonly Rect[], source: Rect, target: Rect): Point[] | undefined {
  const local = routeWithin(obstacles, union([source, target], REACH), source, target);
  if (local) return local;
  return routeWithin(obstacles, union([source, target, ...obstacles], REACH), source, target);
}

/**
 * Edge paths for one laid-out canvas. Lines run along the gutters between
 * cards with rounded corners and meet each card square-on. Returns undefined
 * when one rectangle contains the other (for example a resource and its own
 * VPC): containment is already shown by the grouping itself.
 */
export function createEdgeRouter(obstacles: readonly Rect[]): (source: Rect, target: Rect) => string | undefined {
  return (source, target) => {
    if (contains(source, target) || contains(target, source)) return undefined;
    const points = routeOrthogonal(obstacles, source, target);
    return points ? roundedPath(points) : curvePath(source, target);
  };
}
