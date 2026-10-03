import type { Group, Relationship, Resource } from "../report/schema.gen";
import { relationshipInView, resourceInView, type ReportIndex, type ViewMode } from "../report/view";

/** What each canvas element represents. */
export type CanvasItem =
  | { kind: "group"; group: Group }
  | { kind: "resource"; resource: Resource }
  | { kind: "proxy"; resource: Resource; relationship: Relationship };

export interface CanvasModelEdge {
  id: string;
  source: string;
  target: string;
  relationship: Relationship;
}

/** View-specific structure, independent of geometry. */
export interface CanvasModel {
  items: ReadonlyMap<string, CanvasItem>;
  /** Child group ids per group; "" holds top-level groups. */
  childGroups: ReadonlyMap<string, readonly string[]>;
  /** Resource and proxy ids per group; "" holds unplaced leaves. */
  members: ReadonlyMap<string, readonly string[]>;
  edges: readonly CanvasModelEdge[];
}

/**
 * Builds the architecture view for one side of the change. Resources that
 * define a group (a VPC or subnet) are drawn as that group; association
 * resources become edges. A resource in several subnets appears once, with a
 * small proxy marker in each other subnet instead of a duplicate card.
 */
export function buildCanvasModel(index: ReportIndex, view: ViewMode): CanvasModel {
  const { report, groupOfResource } = index;
  const items = new Map<string, CanvasItem>();
  const members = new Map<string, string[]>();
  const push = (map: Map<string, string[]>, key: string, id: string) => {
    const list = map.get(key);
    if (list) list.push(id);
    else map.set(key, [id]);
  };

  const rels = report.relationships.filter((rel) => relationshipInView(rel, view));
  const resources = report.resources.filter(
    (r) => resourceInView(r, view) && r.role === "entity" && !groupOfResource.has(r.id),
  );
  const visible = new Set(resources.map((r) => r.id));
  const groups = report.groups.filter((g) => g.view === "architecture");
  const groupById = new Map(groups.map((g) => [g.id, g]));
  const placement = new Map(resources.map((r) => [r.id, effectivePlacement(r, rels, groupOfResource, groupById)]));

  for (const resource of resources) {
    items.set(resource.id, { kind: "resource", resource });
    push(members, placement.get(resource.id) ?? "", resource.id);
  }

  const edges: CanvasModelEdge[] = [];
  for (const rel of rels) {
    const source = endpoint(rel.source, visible, groupOfResource);
    const target = endpoint(rel.target, visible, groupOfResource);
    if (!source || !target || source === target) continue;
    const targetGroup = groupOfResource.get(rel.target);
    if (rel.type === "subnet_membership" && targetGroup && visible.has(rel.source)) {
      if (placement.get(rel.source) !== targetGroup) {
        const proxyId = `p-${rel.id}`;
        items.set(proxyId, { kind: "proxy", resource: index.resources.get(rel.source) as Resource, relationship: rel });
        push(members, targetGroup, proxyId);
      }
      continue;
    }
    edges.push({ id: rel.id, source, target, relationship: rel });
  }

  const childGroups = new Map<string, string[]>();
  for (const group of groups) {
    if (group.kind === "unplaced" && !members.has(group.id)) continue;
    items.set(group.id, { kind: "group", group });
    push(childGroups, group.parent && groupById.has(group.parent) ? group.parent : "", group.id);
  }
  return { items, childGroups, members, edges };
}

function endpoint(resourceId: string, visible: ReadonlySet<string>, groupOfResource: ReadonlyMap<string, string>) {
  if (visible.has(resourceId)) return resourceId;
  return groupOfResource.get(resourceId);
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
