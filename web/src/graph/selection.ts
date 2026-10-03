import type { ReportIndex } from "../report/view";
import type { CanvasState } from "./canvasState";

/** Canvas element ids that represent a resource: its card, group or connectors. */
export function canvasIdsFor(index: ReportIndex, resourceId: string): string[] {
  const group = index.groupOfResource.get(resourceId);
  if (group) return [group];
  const resource = index.resources.get(resourceId);
  if (resource?.role !== "association") return [resourceId];
  // An association has no card; focus the resources it connects.
  const ids = new Set<string>();
  for (const rel of index.relationshipsOf.get(resourceId) ?? []) {
    if (rel.evidence.via !== resourceId) continue;
    for (const end of [rel.source, rel.target]) ids.add(index.groupOfResource.get(end) ?? end);
  }
  return [...ids];
}

/** Neighbourhood used to emphasise a selection and trace its relationships. */
export function selectionState(index: ReportIndex, selectedId: string | undefined): CanvasState {
  if (selectedId === undefined) return { selectedId, related: new Set(), tracedEdges: new Set() };
  const related = new Set<string>([selectedId, ...canvasIdsFor(index, selectedId)]);
  const tracedEdges = new Set<string>();
  for (const rel of index.relationshipsOf.get(selectedId) ?? []) {
    tracedEdges.add(rel.id);
    related.add(`p-${rel.id}`);
    for (const end of [rel.source, rel.target, rel.evidence.via]) {
      if (end) related.add(end);
      const group = end ? index.groupOfResource.get(end) : undefined;
      if (group) related.add(group);
    }
  }
  return { selectedId, related, tracedEdges };
}
