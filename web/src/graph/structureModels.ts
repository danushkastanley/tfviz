import type { Group, Relationship, Resource } from "../report/schema.gen";
import { relationshipInView, resourceInView, type ReportIndex, type ViewMode } from "../report/view";
import type { CanvasModelEdge, OpenModel } from "./canvasModel";

function push(map: Map<string, string[]>, key: string, id: string) {
  const list = map.get(key);
  if (list) list.push(id);
  else map.set(key, [id]);
}

/**
 * Architecture view. Resources that define a group (a VPC or subnet) are
 * drawn as that group; association resources become edges. A resource in
 * several subnets appears once, with a proxy marker in each other subnet.
 */
export function architectureModel(index: ReportIndex, view: ViewMode): OpenModel {
  const { report, groupOfResource } = index;
  const model: OpenModel = { placed: new Map(), items: new Map(), childGroups: new Map(), members: new Map(), edges: [], groupOfResource };
  const rels = report.relationships.filter((rel) => relationshipInView(rel, view));
  const resources = report.resources.filter((r) => resourceInView(r, view) && r.role === "entity" && !groupOfResource.has(r.id));
  const visible = new Set(resources.map((r) => r.id));
  const groups = report.groups.filter((g) => g.view === "architecture");
  const groupById = new Map(groups.map((g) => [g.id, g]));
  const placement = new Map(resources.map((r) => [r.id, effectivePlacement(r, rels, groupOfResource, groupById)]));

  for (const resource of resources) {
    model.items.set(resource.id, { kind: "resource", resource });
    push(model.members, placement.get(resource.id) ?? "", resource.id);
  }
  for (const resource of report.resources.filter((r) => resourceInView(r, view))) {
    place(model.placed, placement.get(resource.id) ?? resource.groups.architecture, resource);
  }
  for (const rel of rels) {
    const source = visible.has(rel.source) ? rel.source : groupOfResource.get(rel.source);
    const target = visible.has(rel.target) ? rel.target : groupOfResource.get(rel.target);
    if (!source || !target || source === target) continue;
    const targetGroup = groupOfResource.get(rel.target);
    if (rel.type === "subnet_membership" && targetGroup && visible.has(rel.source)) {
      if (placement.get(rel.source) !== targetGroup) {
        const proxyId = `p-${rel.id}`;
        model.items.set(proxyId, { kind: "proxy", resource: index.resources.get(rel.source) as Resource, relationship: rel });
        push(model.members, targetGroup, proxyId);
      }
      continue;
    }
    model.edges.push(edge(rel, source, target));
  }
  addGroups(model, groups, groupById);
  return model;
}

/**
 * Module view: every resource is a card inside its module instance, with the
 * same identities as the architecture view. Association resources remain
 * edges; subnet membership is drawn as edges to subnet cards.
 */
export function moduleModel(index: ReportIndex, view: ViewMode): OpenModel {
  const { report } = index;
  const model: OpenModel = { placed: new Map(), items: new Map(), childGroups: new Map(), members: new Map(), edges: [], groupOfResource: new Map() };
  const resources = report.resources.filter((r) => resourceInView(r, view) && r.role === "entity");
  const visible = new Set(resources.map((r) => r.id));
  for (const resource of resources) {
    model.items.set(resource.id, { kind: "resource", resource });
    push(model.members, resource.groups.modules, resource.id);
  }
  for (const resource of report.resources.filter((r) => resourceInView(r, view))) {
    place(model.placed, resource.groups.modules, resource);
  }
  for (const rel of report.relationships) {
    if (relationshipInView(rel, view) && visible.has(rel.source) && visible.has(rel.target) && rel.source !== rel.target) {
      model.edges.push(edge(rel, rel.source, rel.target));
    }
  }
  const groups = report.groups.filter((g) => g.view === "modules");
  addGroups(model, groups, new Map(groups.map((g) => [g.id, g])));
  return model;
}

function place(map: Map<string, Resource[]>, key: string, resource: Resource) {
  const list = map.get(key);
  if (list) list.push(resource);
  else map.set(key, [resource]);
}

function edge(rel: Relationship, source: string, target: string): CanvasModelEdge {
  return { id: rel.id, source, target, relationships: [rel] };
}

function addGroups(model: OpenModel, groups: readonly Group[], groupById: ReadonlyMap<string, Group>) {
  for (const group of groups) {
    if (group.kind === "unplaced" && !model.members.has(group.id)) continue;
    model.items.set(group.id, { kind: "group", group });
    push(model.childGroups, group.parent && groupById.has(group.parent) ? group.parent : "", group.id);
  }
}

/**
 * A single-subnet resource that moves between subnets is placed in the
 * subnet it occupies in this view, not always in its post-change subnet.
 */
function effectivePlacement(
  resource: Resource,
  rels: readonly Relationship[],
  groupOfResource: ReadonlyMap<string, string>,
  groupById: ReadonlyMap<string, Group>,
): string {
  const primary = resource.groups.architecture;
  if (groupById.get(primary)?.kind !== "subnet") return primary;
  const subnets = rels
    .filter((rel) => rel.source === resource.id && rel.type === "subnet_membership")
    .map((rel) => groupOfResource.get(rel.target))
    .filter((g): g is string => g !== undefined);
  if (subnets.includes(primary) || subnets.length !== 1) return primary;
  return subnets[0] as string;
}
