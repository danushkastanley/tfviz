import type { NodeProps } from "@xyflow/react";
import { FamilyIcon } from "../icons/FamilyIcon";
import { emphasisClass, useCanvasState } from "./canvasState";
import type { CanvasNode } from "./layout";

const PRESENCE_NOTE = { before: "before change", after: "after change", both: "" } as const;

/**
 * Marks membership of another subnet without duplicating the resource, so
 * counts stay honest. Selecting it selects the resource itself.
 */
export function ProxyNode({ data }: NodeProps<CanvasNode>) {
  const state = useCanvasState();
  if (data.item.kind !== "proxy") return null;
  const { resource, relationship } = data.item;
  const note = PRESENCE_NOTE[relationship.presence];
  return (
    <div className={`proxy-node presence-${relationship.presence} ${emphasisClass(state, [resource.id])}`}>
      <FamilyIcon family={resource.family} size={13} />
      <span className="proxy-node__label">{resource.label}</span>
      {note && <span className="proxy-node__note">{note}</span>}
    </div>
  );
}
