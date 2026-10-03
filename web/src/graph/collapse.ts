import type { CanvasModel, CanvasModelEdge, CollapsedSummary, OpenModel } from "./canvasModel";

/**
 * Hides the contents of collapsed groups. A collapsed group shows how many
 * resources it holds and how many change; relationships that cross its
 * boundary are redrawn to the group, merged when they share endpoints.
 */
export function applyCollapse(open: OpenModel, collapsed: ReadonlySet<string>): CanvasModel {
  const representative = new Map<string, string>();
  const items = new Map(open.items);
  const members = new Map(open.members);
  const childGroups = new Map(open.childGroups);

  // Outermost collapsed groups win: walk from the roots down.
  const visit = (groupId: string, hiddenBy: string | undefined) => {
    const hide = hiddenBy ?? (collapsed.has(groupId) && items.has(groupId) ? groupId : undefined);
    for (const child of open.childGroups.get(groupId) ?? []) {
      if (hide) {
        representative.set(child, hide);
        items.delete(child);
      }
      visit(child, hide);
    }
    for (const leaf of open.members.get(groupId) ?? []) {
      if (hide) {
        representative.set(leaf, hide);
        items.delete(leaf);
      }
    }
    if (hide) {
      members.delete(groupId);
      childGroups.delete(groupId);
    }
  };
  for (const root of open.childGroups.get("") ?? []) visit(root, undefined);

  for (const groupId of collapsed) {
    const item = items.get(groupId);
    if (item?.kind === "group" && !representative.has(groupId)) {
      items.set(groupId, { ...item, collapsed: summarise(open, groupId) });
    }
  }
  // Group-defining resources hidden inside a collapsed group map to it too.
  for (const [resourceId, groupId] of open.groupOfResource) {
    const hidden = representative.get(groupId);
    if (hidden) representative.set(resourceId, hidden);
  }

  const merged = new Map<string, CanvasModelEdge>();
  for (const e of open.edges) {
    const source = representative.get(e.source) ?? e.source;
    const target = representative.get(e.target) ?? e.target;
    if (source === target) continue;
    const key = `${source}→${target}`;
    const existing = merged.get(key);
    if (existing) existing.relationships.push(...e.relationships);
    else merged.set(key, { id: e.id, source, target, relationships: [...e.relationships] });
  }
  return { items, members, childGroups, edges: [...merged.values()], representative, groupOfResource: open.groupOfResource };
}

/** Counts every resource placed anywhere inside a group, of any role. */
function summarise(open: OpenModel, groupId: string): CollapsedSummary {
  const summary: CollapsedSummary = { resources: 0, changed: 0 };
  const walk = (id: string) => {
    for (const resource of open.placed.get(id) ?? []) {
      summary.resources += 1;
      if (resource.change.action !== "no_op") summary.changed += 1;
    }
    for (const child of open.childGroups.get(id) ?? []) walk(child);
  };
  walk(groupId);
  return summary;
}
