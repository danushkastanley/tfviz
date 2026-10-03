import { describe, expect, it } from "vitest";
import { indexReport } from "../report/view";
import { groupId, idOf, loadSample } from "../test/sample";
import { buildCanvasModel, defaultCollapsed } from "./canvasModel";
import { ancestorGroups, canvasIdsFor } from "./selection";
import { largeReport } from "../../e2e/largeReport";

const report = loadSample();
const index = indexReport(report);
const id = (address: string) => idOf(report, address);
const group = (label: string) => groupId(report, label);

describe("module view", () => {
  const model = buildCanvasModel(index, "changes", "modules");

  it("draws VPCs and subnets as cards inside their module", () => {
    const root = report.groups.find((g) => g.view === "modules" && g.label === "Root module")?.id as string;
    expect(model.members.get(root)).toContain(id("aws_vpc.main"));
    expect(model.members.get(root)).toContain(id('aws_subnet.private["a"]'));
    const streaming = report.groups.find((g) => g.label === "module.streaming")?.id as string;
    expect(model.members.get(streaming)).toContain(id("module.streaming.aws_msk_cluster.events"));
    expect(model.childGroups.get(root)).toContain(streaming);
  });

  it("keeps the same identities and draws subnet membership as edges", () => {
    const msk = id("module.streaming.aws_msk_cluster.events");
    const memberships = model.edges.filter((e) => e.source === msk && e.relationships[0]?.type === "subnet_membership");
    expect(memberships).toHaveLength(3);
  });
});

describe("collapse", () => {
  const vpc = group("review-platform");
  const model = buildCanvasModel(index, "changes", "architecture", new Set([vpc]));

  it("hides everything inside a collapsed group and summarises it", () => {
    const item = model.items.get(vpc);
    const inside = report.resources.filter((r) =>
      ancestorGroups(index, r.id, "architecture").includes(vpc) && !(index.groupOfResource.get(r.id) === vpc),
    );
    expect(item?.kind === "group" && item.collapsed).toEqual({
      resources: inside.length,
      changed: inside.filter((r) => r.change.action !== "no_op").length,
    });
    expect(inside.length).toBeGreaterThan(19); // includes association resources, not just cards
    expect(model.items.has(id("module.streaming.aws_msk_cluster.events"))).toBe(false);
    expect(model.items.has(group("private-a"))).toBe(false);
    expect(model.childGroups.has(vpc)).toBe(false);
  });

  it("redraws crossing relationships to the collapsed group and merges them", () => {
    const msk = id("module.streaming.aws_msk_cluster.events");
    expect(model.representative.get(msk)).toBe(vpc);
    expect(model.representative.get(id('aws_subnet.public["a"]'))).toBe(vpc);
    const toKms = model.edges.filter((e) => e.source === vpc && e.target === id("module.streaming.aws_kms_key.msk"));
    expect(toKms).toHaveLength(1);
    expect(model.edges.every((e) => e.source !== e.target)).toBe(true);
  });

  it("lets the outermost collapsed group win", () => {
    const nested = buildCanvasModel(index, "changes", "architecture", new Set([vpc, group("private-a")]));
    expect(nested.representative.get(group("private-a"))).toBe(vpc);
  });

  it("maps a hidden resource to the group that shows it", () => {
    const lookup = { groupOfResource: model.groupOfResource, representative: model.representative };
    expect(canvasIdsFor(index, id("aws_db_instance.orders"), lookup)).toEqual([vpc]);
  });
});

describe("default collapse", () => {
  it("collapses nothing in small reports", () => {
    expect(defaultCollapsed(index, "architecture").size).toBe(0);
  });

  it("collapses large groups in large reports, never accounts or regions", () => {
    const large = indexReport(largeReport(report));
    const collapsed = defaultCollapsed(large, "architecture");
    expect(collapsed.size).toBeGreaterThan(0);
    for (const g of large.report.groups) {
      if (collapsed.has(g.id)) expect(["account", "region"]).not.toContain(g.kind);
    }
  });
});

describe("ancestorGroups", () => {
  it("lists the groups to expand to reveal a resource, innermost first", () => {
    const nat = id("aws_nat_gateway.main");
    expect(ancestorGroups(index, nat, "architecture").slice(0, 2)).toEqual([group("public-b"), group("review-platform")]);
    const streaming = report.groups.find((g) => g.label === "module.streaming")?.id;
    expect(ancestorGroups(index, id("module.streaming.aws_kms_key.msk"), "modules")[0]).toBe(streaming);
  });
});
