import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ResourceIcon } from "../icons/ResourceIcon";
import { presentChange } from "../theme/changes";
import { emphasisClass, useCanvasState } from "./canvasState";
import type { CanvasNode } from "./layout";
import { ChangeBadge } from "./ChangeBadge";

export function ResourceNode({ data }: NodeProps<CanvasNode>) {
  const state = useCanvasState();
  if (data.item.kind !== "resource") return null;
  const { resource } = data.item;
  const change = presentChange(resource.change.action);
  const classes = [
    "resource-card",
    `tone-${change.tone}`,
    resource.support === "generic" ? "is-generic" : "",
    resource.mode === "data" ? "is-data" : "",
    emphasisClass(state, [resource.id]),
  ].join(" ");

  return (
    <div className={classes} data-resource-id={resource.id}>
      <Handle type="target" position={Position.Left} isConnectable={false} />
      <span className="resource-card__icon">
        <ResourceIcon resource={resource} />
      </span>
      <span className="resource-card__text">
        <span className="resource-card__label">{resource.label}</span>
        <span className="resource-card__type">{resource.type}</span>
      </span>
      <ChangeBadge action={resource.change.action} compact />
      <Handle type="source" position={Position.Right} isConnectable={false} />
    </div>
  );
}
