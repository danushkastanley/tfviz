import { Handle, Position, type NodeProps } from "@xyflow/react";
import { emphasisClass, useCanvasState } from "./canvasState";
import type { CanvasNode } from "./layout";
import { ChangeBadge } from "./ChangeBadge";
import { useReportIndex } from "../app/reportContext";

const KIND_LABEL: Record<string, string> = {
  account: "Account",
  region: "Region",
  vpc: "VPC",
  availability_zone: "Availability zone",
  subnet: "Subnet",
  regional_services: "Regional",
  global_services: "Global",
  unplaced: "Unplaced",
  module: "Module",
};

export function GroupNode({ data }: NodeProps<CanvasNode>) {
  const state = useCanvasState();
  const index = useReportIndex();
  if (data.item.kind !== "group") return null;
  const { group } = data.item;
  const resource = group.resource ? index.resources.get(group.resource) : undefined;
  const ids = resource ? [group.id, resource.id] : [group.id];
  const classes = [
    "group-node",
    `group-${group.kind}`,
    group.classification ? `subnet-${group.classification}` : "",
    group.placement === "unresolved" ? "is-unresolved" : "",
    resource ? "is-resource" : "",
    resource ? emphasisClass(state, ids) : "",
  ].join(" ");

  return (
    <div className={classes} data-group-id={group.id}>
      <Handle type="target" position={Position.Left} isConnectable={false} />
      <div className="group-node__header">
        {/* Subnets carry a public/private pill instead, leaving room for the name. */}
        {group.kind !== "subnet" && <span className="group-node__kind">{KIND_LABEL[group.kind] ?? group.kind}</span>}
        <span className="group-node__label">{group.label}</span>
        {group.classification && group.classification !== "unknown" && (
          <span className="group-node__classification">{group.classification}</span>
        )}
        {resource && <ChangeBadge action={resource.change.action} compact />}
      </div>
      <Handle type="source" position={Position.Right} isConnectable={false} />
    </div>
  );
}
