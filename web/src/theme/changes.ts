import type { Change } from "../report/schema.gen";

export type Action = Change["action"];

export interface ChangePresentation {
  /** Short visible label; colour is never the only signal. */
  label: string;
  /** Monochrome symbol shown alongside the label. */
  symbol: string;
  /** CSS custom-property stem, e.g. "create" → var(--change-create). */
  tone: "create" | "update" | "delete" | "replace" | "read" | "none";
}

const PRESENTATION: Record<Action, ChangePresentation> = {
  create: { label: "Added", symbol: "+", tone: "create" },
  update: { label: "Modified", symbol: "~", tone: "update" },
  delete: { label: "Destroyed", symbol: "−", tone: "delete" },
  replace: { label: "Replaced", symbol: "±", tone: "replace" },
  read: { label: "Read during apply", symbol: "↻", tone: "read" },
  // Not a destruction: the infrastructure stays, Terraform stops managing it.
  forget: { label: "Removed from state (not destroyed)", symbol: "⊘", tone: "read" },
  no_op: { label: "Unchanged", symbol: "", tone: "none" },
  unsupported: { label: "Unsupported action", symbol: "?", tone: "none" },
};

export function presentChange(action: Action): ChangePresentation {
  return PRESENTATION[action];
}

export function describeReplaceOrder(order: Change["replace_order"]): string | undefined {
  switch (order) {
    case "delete_first":
      return "Destroyed first, then created. Expect a gap while it is replaced.";
    case "create_first":
      return "Created first, then the old one is destroyed.";
    default:
      return undefined;
  }
}

/** Order used for change filters and the resource list. */
export const ACTION_ORDER: readonly Action[] = ["replace", "delete", "create", "update", "forget", "read", "unsupported", "no_op"];
