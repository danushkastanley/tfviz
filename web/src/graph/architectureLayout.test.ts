import { describe, expect, it } from "vitest";
import { indexReport } from "../report/view";
import { loadSample } from "../test/sample";
import { buildCanvasModel } from "./canvasModel";
import { layoutArchitecture, type Placed } from "./architectureLayout";
import { edgePath } from "./edgePath";

const index = indexReport(loadSample());
const layout = (view: "changes" | "before" | "after" = "changes") => layoutArchitecture(buildCanvasModel(index, view), index);

function overlaps(a: Placed, b: Placed): boolean {
  return a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height;
}

describe("layoutArchitecture", () => {
  it("is deterministic", () => {
    expect(layout()).toEqual(layout());
  });

  it("emits parents before their children", () => {
    const seen = new Set<string>();
    for (const p of layout()) {
      if (p.parent) expect(seen.has(p.parent), p.id).toBe(true);
      seen.add(p.id);
    }
  });

  it("never overlaps siblings and keeps children inside their parent", () => {
    for (const view of ["changes", "before", "after"] as const) {
      const placed = layout(view);
      const byId = new Map(placed.map((p) => [p.id, p]));
      const siblings = new Map<string, Placed[]>();
      for (const p of placed) siblings.set(p.parent ?? "", [...(siblings.get(p.parent ?? "") ?? []), p]);
      for (const group of siblings.values()) {
        for (let i = 0; i < group.length; i++)
          for (let j = i + 1; j < group.length; j++) expect(overlaps(group[i] as Placed, group[j] as Placed), `${group[i]?.id}/${group[j]?.id}`).toBe(false);
      }
      for (const p of placed) {
        const parent = p.parent ? byId.get(p.parent) : undefined;
        if (!parent) continue;
        expect(p.x >= 0 && p.y >= 0 && p.x + p.width <= parent.width && p.y + p.height <= parent.height, p.id).toBe(true);
      }
    }
  });

  it("arranges subnets by availability zone columns with public above private", () => {
    const byId = new Map(layout().map((p) => [p.id, p]));
    const at = (id: string) => byId.get(id) as Placed;
    const [pubA, pubB, pubC, privA] = [at("g4"), at("g5"), at("g6"), at("g7")];
    expect(pubA.x).toBeLessThan(pubB.x);
    expect(pubB.x).toBeLessThan(pubC.x);
    expect(privA.x).toBe(pubA.x);
    expect(privA.y).toBeGreaterThan(pubA.y);
  });
});

describe("edgePath", () => {
  it("draws nothing between a group and something it contains", () => {
    expect(edgePath({ x: 0, y: 0, width: 500, height: 500 }, { x: 10, y: 10, width: 50, height: 50 })).toBeUndefined();
  });

  it("connects facing sides", () => {
    expect(edgePath({ x: 0, y: 0, width: 100, height: 50 }, { x: 0, y: 300, width: 100, height: 50 })).toMatch(/^M 50 50 C/);
  });
});
