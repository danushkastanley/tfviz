export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Point {
  x: number;
  y: number;
}

interface Anchor extends Point {
  /** Unit direction leaving the rectangle. */
  nx: number;
  ny: number;
}

const CORNER_RADIUS = 8;

export function contains(outer: Rect, inner: Rect): boolean {
  return (
    inner.x >= outer.x &&
    inner.y >= outer.y &&
    inner.x + inner.width <= outer.x + outer.width &&
    inner.y + inner.height <= outer.y + outer.height
  );
}

const round = (n: number) => Math.round(n * 10) / 10;

function anchor(rect: Rect, toward: Point, vertical: boolean): Anchor {
  const cx = rect.x + rect.width / 2;
  const cy = rect.y + rect.height / 2;
  if (vertical) {
    const down = toward.y > cy;
    return { x: cx, y: down ? rect.y + rect.height : rect.y, nx: 0, ny: down ? 1 : -1 };
  }
  const right = toward.x > cx;
  return { x: right ? rect.x + rect.width : rect.x, y: cy, nx: right ? 1 : -1, ny: 0 };
}

/**
 * A smooth curve between the facing sides of two rectangles. The router
 * falls back to this only when no clear orthogonal route exists.
 */
export function curvePath(source: Rect, target: Rect): string {
  const sc = { x: source.x + source.width / 2, y: source.y + source.height / 2 };
  const tc = { x: target.x + target.width / 2, y: target.y + target.height / 2 };
  const vertical = Math.abs(tc.y - sc.y) > Math.abs(tc.x - sc.x) * 0.6;
  const a = anchor(source, tc, vertical);
  const b = anchor(target, sc, vertical);
  const bend = Math.max(36, Math.hypot(b.x - a.x, b.y - a.y) / 3);
  return [
    `M ${round(a.x)} ${round(a.y)}`,
    `C ${round(a.x + a.nx * bend)} ${round(a.y + a.ny * bend)}`,
    `${round(b.x + b.nx * bend)} ${round(b.y + b.ny * bend)}`,
    `${round(b.x)} ${round(b.y)}`,
  ].join(" ");
}

/** Drops repeated points and points in the middle of a straight run. */
export function simplify(points: readonly Point[]): Point[] {
  const out: Point[] = [];
  for (const p of points) {
    const last = out[out.length - 1];
    if (last && last.x === p.x && last.y === p.y) continue;
    const before = out[out.length - 2];
    if (before && last && ((before.x === last.x && last.x === p.x) || (before.y === last.y && last.y === p.y))) {
      out[out.length - 1] = p;
      continue;
    }
    out.push(p);
  }
  return out;
}

/** An orthogonal polyline as an SVG path with softly rounded corners. */
export function roundedPath(points: readonly Point[]): string {
  const pts = simplify(points);
  const first = pts[0];
  if (!first) return "";
  const parts = [`M ${round(first.x)} ${round(first.y)}`];
  for (let i = 1; i < pts.length - 1; i++) {
    const prev = pts[i - 1] as Point;
    const corner = pts[i] as Point;
    const next = pts[i + 1] as Point;
    const inLength = Math.hypot(corner.x - prev.x, corner.y - prev.y);
    const outLength = Math.hypot(next.x - corner.x, next.y - corner.y);
    const r = Math.min(CORNER_RADIUS, inLength / 2, outLength / 2);
    const a = { x: corner.x + ((prev.x - corner.x) / inLength) * r, y: corner.y + ((prev.y - corner.y) / inLength) * r };
    const b = { x: corner.x + ((next.x - corner.x) / outLength) * r, y: corner.y + ((next.y - corner.y) / outLength) * r };
    parts.push(`L ${round(a.x)} ${round(a.y)}`, `Q ${round(corner.x)} ${round(corner.y)} ${round(b.x)} ${round(b.y)}`);
  }
  const last = pts[pts.length - 1] as Point;
  if (pts.length > 1) parts.push(`L ${round(last.x)} ${round(last.y)}`);
  return parts.join(" ");
}
