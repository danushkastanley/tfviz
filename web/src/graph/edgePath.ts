export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

interface Anchor {
  x: number;
  y: number;
  /** Unit direction leaving the rectangle. */
  nx: number;
  ny: number;
}

function contains(outer: Rect, inner: Rect): boolean {
  return (
    inner.x >= outer.x &&
    inner.y >= outer.y &&
    inner.x + inner.width <= outer.x + outer.width &&
    inner.y + inner.height <= outer.y + outer.height
  );
}

function anchor(rect: Rect, toward: { x: number; y: number }, vertical: boolean): Anchor {
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
 * A smooth curve between the facing sides of two rectangles. Returns
 * undefined when one contains the other (for example a resource and its own
 * VPC): containment is already shown by the grouping itself.
 */
export function edgePath(source: Rect, target: Rect): string | undefined {
  if (contains(source, target) || contains(target, source)) return undefined;
  const sc = { x: source.x + source.width / 2, y: source.y + source.height / 2 };
  const tc = { x: target.x + target.width / 2, y: target.y + target.height / 2 };
  const vertical = Math.abs(tc.y - sc.y) > Math.abs(tc.x - sc.x) * 0.6;
  const a = anchor(source, tc, vertical);
  const b = anchor(target, sc, vertical);
  const bend = Math.max(36, Math.hypot(b.x - a.x, b.y - a.y) / 3);
  const round = (n: number) => Math.round(n * 10) / 10;
  return [
    `M ${round(a.x)} ${round(a.y)}`,
    `C ${round(a.x + a.nx * bend)} ${round(a.y + a.ny * bend)}`,
    `${round(b.x + b.nx * bend)} ${round(b.y + b.ny * bend)}`,
    `${round(b.x)} ${round(b.y)}`,
  ].join(" ");
}
