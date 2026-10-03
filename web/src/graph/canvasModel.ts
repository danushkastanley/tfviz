import type { Group, Relationship, Resource } from "../report/schema.gen";
import type { ReportIndex, ViewMode } from "../report/view";
import { applyCollapse } from "./collapse";
import { architectureModel, moduleModel } from "./structureModels";

/** The structure the canvas is organised by. */
export type Structure = "architecture" | "modules";

/** Summary shown on a collapsed group. */
export interface CollapsedSummary {
  resources: number;
  changed: number;
}

/** What each canvas element represents. */
export type CanvasItem =
  | { kind: "group"; group: Group; collapsed?: CollapsedSummary }
  | { kind: "resource"; resource: Resource }
  | { kind: "proxy"; resource: Resource; relationship: Relationship };

/** One drawn edge; several relationships share an edge when groups collapse. */
export interface CanvasModelEdge {
  id: string;
  source: string;
  target: string;
  relationships: Relationship[];
}

/** View-specific structure, independent of geometry. */
export interface CanvasModel {
  items: ReadonlyMap<string, CanvasItem>;
  /** Child group ids per group; "" holds top-level groups. */
  childGroups: ReadonlyMap<string, readonly string[]>;
  /** Resource and proxy ids per group; "" holds unplaced leaves. */
  members: ReadonlyMap<string, readonly string[]>;
  edges: readonly CanvasModelEdge[];
  /** For anything hidden inside a collapsed group: that group's id. */
  representative: ReadonlyMap<string, string>;
  /** Group ids by the resource that defines them (VPCs and subnets). */
  groupOfResource: ReadonlyMap<string, string>;
}

/** Structure before collapsing: the builders produce this. */
export interface OpenModel {
  /** Every resource in this view (any role), by its primary group, for summaries. */
  placed: Map<string, Resource[]>;
  items: Map<string, CanvasItem>;
  childGroups: Map<string, string[]>;
  members: Map<string, string[]>;
  edges: CanvasModelEdge[];
  groupOfResource: ReadonlyMap<string, string>;
}

export function buildCanvasModel(
  index: ReportIndex,
  view: ViewMode,
  structure: Structure = "architecture",
  collapsed: ReadonlySet<string> = new Set(),
): CanvasModel {
  const open = structure === "architecture" ? architectureModel(index, view) : moduleModel(index, view);
  return applyCollapse(open, collapsed);
}

/** Groups whose contents are large enough to start collapsed in big reports. */
export function defaultCollapsed(index: ReportIndex, structure: Structure): Set<string> {
  const open = structure === "architecture" ? architectureModel(index, "changes") : moduleModel(index, "changes");
  const collapsed = new Set<string>();
  if (index.report.resources.length <= 300) return collapsed;
  const size = (id: string): number =>
    (open.members.get(id)?.length ?? 0) + (open.childGroups.get(id) ?? []).reduce((n, child) => n + size(child), 0);
  for (const [id, item] of open.items) {
    if (item.kind === "group" && item.group.kind !== "account" && item.group.kind !== "region" && size(id) > 60) {
      collapsed.add(id);
    }
  }
  return collapsed;
}
