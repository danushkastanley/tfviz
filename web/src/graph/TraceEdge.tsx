import type { EdgeProps } from "@xyflow/react";
import { useCanvasState } from "./canvasState";
import type { CanvasEdge } from "./layout";

/**
 * Recorded relationships. When a related resource is selected, a short pulse
 * traces the association: a relationship trace, not observed traffic.
 */
export function TraceEdge({ data }: EdgeProps<CanvasEdge>) {
  const state = useCanvasState();
  if (!data) return null;
  const traced = data.relationships.some((rel) => state.tracedEdges.has(rel.id));
  const dimmed = state.selectedId !== undefined && !traced;
  const presences = new Set(data.relationships.map((rel) => rel.presence));
  const presence = presences.size === 1 ? [...presences][0] : "both";
  const classes = ["trace-edge", `presence-${presence}`, traced ? "is-traced" : "", dimmed ? "is-dimmed" : ""];
  return (
    <g className={classes.join(" ")}>
      <path className="trace-edge__line" d={data.path} fill="none" />
      {traced && <path className="trace-edge__pulse" d={data.path} fill="none" />}
    </g>
  );
}
