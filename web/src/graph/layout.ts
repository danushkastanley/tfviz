import type { Edge, Node } from "@xyflow/react";
import type { Relationship } from "../report/schema.gen";
import type { ReportIndex, ViewMode } from "../report/view";
import { buildCanvasModel, type CanvasItem, type Structure } from "./canvasModel";
import { layoutArchitecture, type Placed } from "./architectureLayout";
import { edgePath, type Rect } from "./edgePath";
import type { CanvasLookup } from "./selection";

export type CanvasNodeData = { item: CanvasItem };
/** One drawn edge may carry several relationships when groups collapse. */
export type CanvasEdgeData = { relationships: Relationship[]; path: string };
export type CanvasNode = Node<CanvasNodeData>;
export type CanvasEdge = Edge<CanvasEdgeData>;

export interface CanvasLayout {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  lookup: CanvasLookup;
  /** Milliseconds spent computing the layout, for the performance budget. */
  layoutMs: number;
}

export function layoutCanvas(
  index: ReportIndex,
  view: ViewMode,
  structure: Structure,
  collapsed: ReadonlySet<string>,
): CanvasLayout {
  const started = performance.now();
  const model = buildCanvasModel(index, view, structure, collapsed);
  const placed = layoutArchitecture(model, index);

  const absolute = new Map<string, Rect>();
  const nodes: CanvasNode[] = [];
  // The layout emits parents before children, as React Flow requires.
  for (const p of placed) {
    const item = model.items.get(p.id);
    if (!item) continue;
    const origin = p.parent ? absolute.get(p.parent) : undefined;
    absolute.set(p.id, { x: (origin?.x ?? 0) + p.x, y: (origin?.y ?? 0) + p.y, width: p.width, height: p.height });
    nodes.push(toNode(p, item));
  }

  const edges: CanvasEdge[] = [];
  for (const edge of model.edges) {
    const source = absolute.get(edge.source);
    const target = absolute.get(edge.target);
    const path = source && target ? edgePath(source, target) : undefined;
    if (!path) continue;
    edges.push({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      type: "trace",
      data: { relationships: edge.relationships, path },
      zIndex: 2,
      selectable: false,
    });
  }
  const lookup = { groupOfResource: model.groupOfResource, representative: model.representative };
  return { nodes, edges, lookup, layoutMs: performance.now() - started };
}

function toNode(p: Placed, item: CanvasItem): CanvasNode {
  return {
    id: p.id,
    type: item.kind,
    position: { x: p.x, y: p.y },
    data: { item },
    ...(p.parent ? { parentId: p.parent } : {}),
    width: p.width,
    height: p.height,
    style: { width: p.width, height: p.height },
    draggable: false,
    connectable: false,
    selectable: item.kind !== "proxy",
    zIndex: item.kind === "group" ? 0 : 1,
  };
}
