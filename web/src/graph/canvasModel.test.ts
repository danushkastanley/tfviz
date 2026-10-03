import { describe, expect, it } from "vitest";
import { indexReport } from "../report/view";
import { groupId, idOf, loadSample } from "../test/sample";
import { buildCanvasModel } from "./canvasModel";

const report = loadSample();
const index = indexReport(report);
const id = (address: string) => idOf(report, address);
const group = (label: string) => groupId(report, label);
const groupOf = (model: ReturnType<typeof buildCanvasModel>, itemId: string) =>
  [...model.members].find(([, ids]) => ids.includes(itemId))?.[0];
const proxiesOf = (model: ReturnType<typeof buildCanvasModel>, resourceId: string) =>
  [...model.items.entries()].filter(([key, item]) => key.startsWith("p-") && item.kind === "proxy" && item.resource.id === resourceId).map(([key]) => key);

describe("buildCanvasModel", () => {
  it("draws every entity resource once, and never association resources or group-defining resources as cards", () => {
    const model = buildCanvasModel(index, "changes");
    const cards = [...model.members.values()].flat().filter((key) => !key.startsWith("p-"));
    expect(new Set(cards).size).toBe(cards.length);
    for (const resource of report.resources) {
      const expected = resource.role === "entity" && !index.groupOfResource.has(resource.id);
      expect(cards.includes(resource.id), resource.address).toBe(expected);
    }
  });

  it("marks multi-subnet membership with proxies instead of duplicate cards", () => {
    const model = buildCanvasModel(index, "changes");
    const msk = id("module.streaming.aws_msk_cluster.events");
    expect(proxiesOf(model, msk).map((p) => groupOf(model, p)).sort()).toEqual([group("private-a"), group("private-b"), group("private-c")].sort());
    expect(groupOf(model, msk)).toBe(group("review-platform"));
  });

  it("places a moved NAT gateway in the subnet it occupies in each view", () => {
    const nat = id("aws_nat_gateway.main");
    const changes = buildCanvasModel(index, "changes");
    expect(groupOf(changes, nat)).toBe(group("public-b"));
    expect(proxiesOf(changes, nat).map((p) => groupOf(changes, p))).toEqual([group("public-a")]);

    const before = buildCanvasModel(index, "before");
    expect(groupOf(before, nat)).toBe(group("public-a"));
    expect(proxiesOf(before, nat)).toEqual([]);

    expect(groupOf(buildCanvasModel(index, "after"), nat)).toBe(group("public-b"));
  });

  it("omits created resources and after-only relationships from the before view", () => {
    const model = buildCanvasModel(index, "before");
    expect(model.items.has(id("module.streaming.aws_secretsmanager_secret.orders_scram[0]"))).toBe(false);
    expect(model.edges.every((e) => e.relationship.presence !== "after")).toBe(true);
  });

  it("omits relationships that only existed before from the after view", () => {
    const model = buildCanvasModel(index, "after");
    expect(model.edges.every((e) => e.relationship.presence !== "before")).toBe(true);
  });

  it("draws association-backed relationships as edges between the resources they connect", () => {
    const model = buildCanvasModel(index, "changes");
    const scram = model.edges.find((e) => e.relationship.evidence.via === id("module.streaming.aws_msk_scram_secret_association.events[0]"));
    expect(scram).toMatchObject({
      source: id("module.streaming.aws_msk_cluster.events"),
      target: id("module.streaming.aws_secretsmanager_secret.orders_scram[0]"),
    });
    const route = model.edges.find((e) => e.relationship.evidence.via === id('aws_route_table_association.public["a"]'));
    expect(route).toMatchObject({ source: id("aws_route_table.public"), target: group("public-a") });
  });
});
