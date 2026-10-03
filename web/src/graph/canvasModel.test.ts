import { describe, expect, it } from "vitest";
import { indexReport } from "../report/view";
import { loadSample } from "../test/sample";
import { buildCanvasModel } from "./canvasModel";

const index = indexReport(loadSample());
const groupOf = (model: ReturnType<typeof buildCanvasModel>, id: string) =>
  [...model.members].find(([, ids]) => ids.includes(id))?.[0];

describe("buildCanvasModel", () => {
  it("draws every entity resource once, and never association resources or group-defining resources as cards", () => {
    const model = buildCanvasModel(index, "changes");
    const cards = [...model.members.values()].flat().filter((id) => !id.startsWith("p-"));
    expect(new Set(cards).size).toBe(cards.length);
    for (const resource of index.report.resources) {
      const expected = resource.role === "entity" && !index.groupOfResource.has(resource.id);
      expect(cards.includes(resource.id), resource.address).toBe(expected);
    }
  });

  it("marks multi-subnet membership with proxies instead of duplicate cards", () => {
    const model = buildCanvasModel(index, "changes");
    const mskProxies = [...model.items.entries()].filter(([id, item]) => id.startsWith("p-") && item.kind === "proxy" && item.resource.id === "r36");
    expect(mskProxies.map(([id]) => groupOf(model, id)).sort()).toEqual(["g7", "g8", "g9"]);
    expect(groupOf(model, "r36")).toBe("g3");
  });

  it("places a moved NAT gateway in the subnet it occupies in each view", () => {
    const changes = buildCanvasModel(index, "changes");
    expect(groupOf(changes, "r10")).toBe("g5");
    const beforeProxy = [...changes.items.entries()].find(([, item]) => item.kind === "proxy" && item.resource.id === "r10");
    expect(beforeProxy && groupOf(changes, beforeProxy[0])).toBe("g4");

    const before = buildCanvasModel(index, "before");
    expect(groupOf(before, "r10")).toBe("g4");
    expect([...before.items.values()].some((item) => item.kind === "proxy" && item.resource.id === "r10")).toBe(false);

    expect(groupOf(buildCanvasModel(index, "after"), "r10")).toBe("g5");
  });

  it("omits created resources and after-only relationships from the before view", () => {
    const model = buildCanvasModel(index, "before");
    expect(model.items.has("r37")).toBe(false);
    expect(model.edges.every((e) => e.relationship.presence !== "after")).toBe(true);
  });

  it("omits relationships that only existed before from the after view", () => {
    const model = buildCanvasModel(index, "after");
    expect(model.edges.every((e) => e.relationship.presence !== "before")).toBe(true);
  });

  it("draws association-backed relationships as edges between the resources they connect", () => {
    const model = buildCanvasModel(index, "changes");
    const scram = model.edges.find((e) => e.relationship.evidence.via === "r39");
    expect(scram).toMatchObject({ source: "r36", target: "r37" });
    const route = model.edges.find((e) => e.relationship.evidence.via === "r13");
    expect(route).toMatchObject({ source: "r11", target: "g4" });
  });
});
