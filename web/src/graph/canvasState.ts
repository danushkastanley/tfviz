import { createContext, useContext } from "react";

/** Selection state shared with canvas nodes without rebuilding the node list. */
export interface CanvasState {
  selectedId: string | undefined;
  /** Canvas element ids related to the selection, including the selection. */
  related: ReadonlySet<string>;
  /** Relationship ids traced from the selection. */
  tracedEdges: ReadonlySet<string>;
}

export const CanvasStateContext = createContext<CanvasState>({
  selectedId: undefined,
  related: new Set(),
  tracedEdges: new Set(),
});

export function useCanvasState(): CanvasState {
  return useContext(CanvasStateContext);
}

export function emphasisClass(state: CanvasState, ids: readonly string[]): string {
  if (state.selectedId === undefined) return "";
  if (ids.includes(state.selectedId)) return "is-selected";
  return ids.some((id) => state.related.has(id)) ? "is-related" : "is-dimmed";
}

/** Canvas actions available to nodes. */
export interface CanvasActions {
  toggleGroup: (groupId: string) => void;
  focusGroup: (groupId: string) => void;
}

export const CanvasActionsContext = createContext<CanvasActions>({ toggleGroup: () => undefined, focusGroup: () => undefined });

export function useCanvasActions(): CanvasActions {
  return useContext(CanvasActionsContext);
}
