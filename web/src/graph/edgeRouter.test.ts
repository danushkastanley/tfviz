import { describe, expect, it } from "vitest";
import { indexReport, type ReportIndex } from "../report/view";
import { loadSample } from "../test/sample";
import { largeReport } from "../../e2e/largeReport";
import { buildCanvasModel } from "./canvasModel";
import { layoutArchitecture } from "./architectureLayout";
import { contains, roundedPath, type Point, type Rect } from "./edgePath";
import { createEdgeRouter, routeOrthogonal } from "./edgeRouter";

const card = (x: number, y: number): Rect => ({ x, y, width: 100, height: 50 });

/** True when the segment a→b passes through the inside of the rectangle. */
function crosses(a: Point, b: Point, r: Rect): boolean {
  const [x0, x1] = [Math.min(a.x, b.x), Math.max(a.x, b.x)];
  const [y0, y1] = [Math.min(a.y, b.y), Math.max(a.y, b.y)];
  return x0 < r.x + r.width && x1 > r.x && y0 < r.y + r.height && y1 > r.y;
}

function segments(points: readonly Point[]): [Point, Point][] {
  return points.slice(1).map((p, i) => [points[i] as Point, p]);
}

/** Every leaf card and every edge, in absolute coordinates, for one view. */
function scene(index: ReportIndex, collapsed: ReadonlySet<string> = new Set()) {
  const model = buildCanvasModel(index, "changes", "architecture", collapsed);
  const absolute = new Map<string, Rect>();
  const cards: Rect[] = [];
  for (const p of layoutArchitecture(model, index)) {
    const origin = p.parent ? absolute.get(p.parent) : undefined;
    const rect = { x: (origin?.x ?? 0) + p.x, y: (origin?.y ?? 0) + p.y, width: p.width, height: p.height };
    absolute.set(p.id, rect);
    const item = model.items.get(p.id);
    if (item && (item.kind !== "group" || item.collapsed)) cards.push(rect);
  }
  const edges = model.edges
    .map((e) => ({ source: absolute.get(e.source) as Rect, target: absolute.get(e.target) as Rect }))
    .filter((e) => !contains(e.source, e.target) && !contains(e.target, e.source));
  return { cards, edges };
}

describe("routeOrthogonal", () => {
  it("joins stacked cards with a straight line between their facing sides", () => {
    expect(routeOrthogonal([], card(0, 0), card(0, 300))).toEqual([
      { x: 50, y: 50 },
      { x: 50, y: 56 },
      { x: 50, y: 294 },
      { x: 50, y: 300 },
    ]);
  });

  it("goes round a card in the way instead of under it", () => {
    const blocker = card(0, 150);
    const route = routeOrthogonal([blocker], card(0, 0), card(0, 300));
    expect(route).toBeDefined();
    for (const [a, b] of segments(route as Point[])) expect(crosses(a, b, blocker)).toBe(false);
  });

  it("uses only horizontal and vertical segments", () => {
    const route = routeOrthogonal([card(150, 100)], card(0, 0), card(300, 220)) as Point[];
    for (const [a, b] of segments(route)) expect(a.x === b.x || a.y === b.y).toBe(true);
  });

  it("looks across the whole canvas when the way round is far away", () => {
    // A tall wall between the cards reaches well beyond the nearby search area.
    const wall = { x: 250, y: -400, width: 50, height: 900 };
    const route = routeOrthogonal([wall], card(0, 0), card(600, 0));
    expect(route).toBeDefined();
    for (const [a, b] of segments(route as Point[])) expect(crosses(a, b, wall)).toBe(false);
  });
});

describe("createEdgeRouter", () => {
  it("draws nothing between a group and something it contains", () => {
    const route = createEdgeRouter([]);
    expect(route({ x: 0, y: 0, width: 500, height: 500 }, { x: 10, y: 10, width: 50, height: 50 })).toBeUndefined();
  });

  it("is deterministic", () => {
    const obstacles = [card(150, 100), card(150, 200)];
    expect(createEdgeRouter(obstacles)(card(0, 0), card(300, 300))).toBe(createEdgeRouter(obstacles)(card(0, 0), card(300, 300)));
  });
});

describe("roundedPath", () => {
  it("rounds each corner without overshooting short segments", () => {
    expect(roundedPath([{ x: 0, y: 0 }, { x: 0, y: 40 }, { x: 6, y: 40 }])).toBe("M 0 0 L 0 37 Q 0 40 3 40 L 6 40");
  });
});

describe("routes on real layouts", () => {
  const sample = indexReport(loadSample());

  it("keeps every relationship line clear of every other card", () => {
    const { cards, edges } = scene(sample);
    expect(edges.length).toBeGreaterThan(10);
    for (const { source, target } of edges) {
      const route = routeOrthogonal(cards, source, target);
      expect(route).toBeDefined();
      const others = cards.filter((c) => c !== source && c !== target);
      for (const [a, b] of segments(route as Point[])) for (const c of others) expect(crosses(a, b, c)).toBe(false);
    }
  });

  it("routes a fully expanded 500-resource stack", () => {
    const { cards, edges } = scene(indexReport(largeReport(loadSample())));
    const route = createEdgeRouter(cards);
    for (const { source, target } of edges) expect(route(source, target)).toMatch(/^M /);
  });
});
