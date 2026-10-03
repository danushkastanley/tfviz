import { useEffect, useMemo, useState } from "react";
import { Background, Controls, Panel, ReactFlow, useReactFlow, type NodeMouseHandler } from "@xyflow/react";
import type { ReportIndex } from "../report/view";
import type { CanvasEdge, CanvasLayout, CanvasNode } from "./layout";
import { ResourceNode } from "./ResourceNode";
import { GroupNode } from "./GroupNode";
import { ProxyNode } from "./ProxyNode";
import { TraceEdge } from "./TraceEdge";
import { canvasIdsFor } from "./selection";
import { CanvasActionsContext, useCanvasState } from "./canvasState";

const nodeTypes = { resource: ResourceNode, group: GroupNode, proxy: ProxyNode };
const edgeTypes = { trace: TraceEdge };

/**
 * Above this many relationships, drawing them all at rest hides the diagram
 * under a haze of lines. Dense views show only the selection's relationships.
 */
export const DENSE_EDGE_LIMIT = 150;

interface CanvasProps {
  index: ReportIndex;
  layout: CanvasLayout;
  /** Remounts the flow (and refits) when the structure or view changes. */
  layoutKey: string;
  /** Changes whenever the navigator asks the canvas to bring a resource into view. */
  focusRequest: { id: string; seq: number } | undefined;
  reducedMotion: boolean;
  onSelect: (resourceId: string | undefined) => void;
  onToggleGroup: (groupId: string) => void;
}

export function Canvas({ index, layout, layoutKey, focusRequest, reducedMotion, onSelect, onToggleGroup }: CanvasProps) {
  const flow = useReactFlow<CanvasNode, CanvasEdge>();
  const { tracedEdges } = useCanvasState();
  const [focusedGroup, setFocusedGroup] = useState<string>();
  const duration = reducedMotion ? 0 : 350;

  const dense = layout.edges.length > DENSE_EDGE_LIMIT;
  const edges = useMemo(
    () => (dense ? layout.edges.filter((e) => e.data?.relationships.some((rel) => tracedEdges.has(rel.id))) : layout.edges),
    [dense, layout.edges, tracedEdges],
  );

  useEffect(() => {
    if (!focusRequest) return;
    // Wait a frame so a path that was just expanded has been laid out.
    const frame = requestAnimationFrame(() => {
      const ids = canvasIdsFor(index, focusRequest.id, layout.lookup).filter((id) => flow.getNode(id));
      if (ids.length > 0) void flow.fitView({ nodes: ids.map((id) => ({ id })), padding: 0.6, maxZoom: 1.25, duration });
    });
    return () => cancelAnimationFrame(frame);
    // Only a new focus request moves the viewport.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focusRequest]);

  const actions = useMemo(
    () => ({
      toggleGroup: onToggleGroup,
      focusGroup: (groupId: string) => {
        setFocusedGroup(groupId);
        void flow.fitView({ nodes: [{ id: groupId }], padding: 0.08, duration });
      },
    }),
    [onToggleGroup, flow, duration],
  );

  const showAll = () => {
    setFocusedGroup(undefined);
    void flow.fitView({ padding: 0.05, duration });
  };

  const handleNodeClick: NodeMouseHandler<CanvasNode> = (_, node) => {
    const item = node.data.item;
    onSelect(item.kind === "group" ? item.group.resource : item.resource.id);
  };

  return (
    <CanvasActionsContext.Provider value={actions}>
      {/* Remounting per layout key lets React Flow fit the new layout once measured. */}
      <ReactFlow
        key={layoutKey}
        fitView
        fitViewOptions={{ padding: 0.05 }}
        nodes={layout.nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodeClick={handleNodeClick}
        onPaneClick={() => onSelect(undefined)}
        nodesDraggable={false}
        nodesConnectable={false}
        edgesFocusable={false}
        nodesFocusable={false}
        zoomOnDoubleClick={false}
        minZoom={0.05}
        maxZoom={2}
        proOptions={{ hideAttribution: true }}
        aria-label="Infrastructure diagram"
      >
        <Background gap={24} size={1} />
        {focusedGroup && <Breadcrumbs index={index} groupId={focusedGroup} onShowAll={showAll} />}
        {dense && (
          <Panel position="top-left" className="canvas-note">
            {layout.edges.length.toLocaleString("en-GB")} relationships are hidden at this size. Select a resource to trace its relationships.
          </Panel>
        )}
        <Controls showInteractive={false} position="bottom-right" />
      </ReactFlow>
    </CanvasActionsContext.Provider>
  );
}

function Breadcrumbs({ index, groupId, onShowAll }: { index: ReportIndex; groupId: string; onShowAll: () => void }) {
  const groups = new Map(index.report.groups.map((g) => [g.id, g]));
  const path: string[] = [];
  for (let g = groups.get(groupId); g; g = g.parent ? groups.get(g.parent) : undefined) path.unshift(g.label);
  return (
    <Panel position="top-center" className="canvas-breadcrumbs">
      <nav aria-label="Focused group">
        <ol>
          {path.map((label, i) => (
            <li key={`${i}-${label}`}>{label}</li>
          ))}
        </ol>
        <button type="button" className="link-button" onClick={onShowAll}>
          Show whole diagram
        </button>
      </nav>
    </Panel>
  );
}
