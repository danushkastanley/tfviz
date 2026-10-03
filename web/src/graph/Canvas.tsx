import { useEffect, useMemo } from "react";
import { Background, Controls, ReactFlow, useReactFlow, type NodeMouseHandler } from "@xyflow/react";
import type { ReportIndex, ViewMode } from "../report/view";
import { layoutCanvas, type CanvasEdge, type CanvasLayout, type CanvasNode } from "./layout";
import { ResourceNode } from "./ResourceNode";
import { GroupNode } from "./GroupNode";
import { ProxyNode } from "./ProxyNode";
import { TraceEdge } from "./TraceEdge";
import { canvasIdsFor } from "./selection";

const nodeTypes = { resource: ResourceNode, group: GroupNode, proxy: ProxyNode };
const edgeTypes = { trace: TraceEdge };

interface CanvasProps {
  index: ReportIndex;
  view: ViewMode;
  /** Changes whenever the navigator asks the canvas to bring a resource into view. */
  focusRequest: { id: string; seq: number } | undefined;
  reducedMotion: boolean;
  onSelect: (resourceId: string | undefined) => void;
  onLayout: (layout: CanvasLayout) => void;
}

export function Canvas({ index, view, focusRequest, reducedMotion, onSelect, onLayout }: CanvasProps) {
  const layout = useMemo(() => layoutCanvas(index, view), [index, view]);
  const flow = useReactFlow<CanvasNode, CanvasEdge>();

  useEffect(() => onLayout(layout), [layout, onLayout]);

  useEffect(() => {
    if (!focusRequest) return;
    const ids = canvasIdsFor(index, focusRequest.id).filter((id) => flow.getNode(id));
    if (ids.length === 0) return;
    void flow.fitView({ nodes: ids.map((id) => ({ id })), padding: 0.6, maxZoom: 1.25, duration: reducedMotion ? 0 : 350 });
    // Only a new focus request moves the viewport.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focusRequest]);

  const handleNodeClick: NodeMouseHandler<CanvasNode> = (_, node) => {
    const item = node.data.item;
    onSelect(item.kind === "group" ? item.group.resource : item.resource.id);
  };

  return (
    // Remounting per view lets React Flow fit the new layout once it is measured.
    <ReactFlow
      key={view}
      fitView
      fitViewOptions={{ padding: 0.05 }}
      nodes={layout.nodes}
      edges={layout.edges}
      nodeTypes={nodeTypes}
      edgeTypes={edgeTypes}
      onNodeClick={handleNodeClick}
      onPaneClick={() => onSelect(undefined)}
      nodesDraggable={false}
      nodesConnectable={false}
      edgesFocusable={false}
      nodesFocusable={false}
      minZoom={0.08}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
      aria-label="Infrastructure diagram"
    >
      <Background gap={24} size={1} />
      <Controls showInteractive={false} position="bottom-right" />
    </ReactFlow>
  );
}
