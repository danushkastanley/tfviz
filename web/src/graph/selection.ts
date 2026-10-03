import type { ReportIndex } from "../report/view";
import type { CanvasState } from "./canvasState";

/** How the current canvas shows resources: which define groups, which are hidden. */
export interface CanvasLookup {
  groupOfResource: ReadonlyMap<string, string>;
  representative: ReadonlyMap<string, string>;
}

const EMPTY: CanvasLookup = { groupOfResource: new Map(), representative: new Map() };

/**
 * Canvas element ids that represent a resource: its card, the group it
 * defines, the resources an association connects, or the collapsed group
 * that currently hides any of these.
 */
export function canvasIdsFor(index: ReportIndex, resourceId: string, lookup: CanvasLookup = EMPTY): string[] {
  const { groupOfResource, representative } = lookup;
  const shown = (id: string) => representative.get(id) ?? id;
  const group = groupOfResource.get(resourceId);
  if (group) return [shown(group)];
  const resource = index.resources.get(resourceId);
  if (resource?.role !== "association") return [shown(resourceId)];
  const ids = new Set<string>();
  for (const rel of index.relationshipsOf.get(resourceId) ?? []) {
    if (rel.evidence.via !== resourceId) continue;
    for (const end of [rel.source, rel.target]) ids.add(shown(groupOfResource.get(end) ?? end));
  }
  return [...ids];
}

/** Neighbourhood used to emphasise a selection and trace its relationships. */
export function selectionState(index: ReportIndex, selectedId: string | undefined, lookup: CanvasLookup = EMPTY): CanvasState {
  if (selectedId === undefined) return { selectedId, related: new Set(), tracedEdges: new Set() };
  const shown = (id: string) => lookup.representative.get(id) ?? id;
  const related = new Set<string>([selectedId, ...canvasIdsFor(index, selectedId, lookup)]);
  const tracedEdges = new Set<string>();
  for (const rel of index.relationshipsOf.get(selectedId) ?? []) {
    tracedEdges.add(rel.id);
    related.add(`p-${rel.id}`);
    for (const end of [rel.source, rel.target, rel.evidence.via]) {
      if (!end) continue;
      related.add(shown(end));
      const group = lookup.groupOfResource.get(end);
      if (group) related.add(shown(group));
    }
  }
  return { selectedId, related, tracedEdges };
}

/** The groups that contain a resource in a structure, innermost first. */
export function ancestorGroups(index: ReportIndex, resourceId: string, structure: "architecture" | "modules"): string[] {
  const resource = index.resources.get(resourceId);
  if (!resource) return [];
  const parents = new Map(index.report.groups.map((g) => [g.id, g.parent]));
  const out: string[] = [];
  let current: string | undefined = structure === "architecture" ? (index.groupOfResource.get(resourceId) ?? resource.groups.architecture) : resource.groups.modules;
  while (current && !out.includes(current)) {
    out.push(current);
    current = parents.get(current);
  }
  return out;
}
