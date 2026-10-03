import type { EdgeProps } from "@xyflow/react";
import { useCanvasState } from "./canvasState";
import type { CanvasEdge } from "./layout";

/**
 * A recorded relationship. When its resource is selected, a short pulse
 * traces the association. It is a relationship trace, not observed traffic.
 */
export function TraceEdge({ id, data }: EdgeProps<CanvasEdge>) {
  const state = useCanvasState();
  if (!data) return null;
  const traced = state.tracedEdges.has(id);
  const dimmed = state.selectedId !== undefined && !traced;
  const classes = ["trace-edge", `presence-${data.relationship.presence}`, traced ? "is-traced" : "", dimmed ? "is-dimmed" : ""];
  return (
    <g className={classes.join(" ")}>
      <path className="trace-edge__line" d={data.path} fill="none" />
      {traced && <path className="trace-edge__pulse" d={data.path} fill="none" />}
    </g>
  );
}
