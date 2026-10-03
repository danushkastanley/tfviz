import { useState } from "react";
import type { ExploreConfig } from "../report/load";

type State = { status: "idle" } | { status: "refreshing" } | { status: "failed"; message: string };

/**
 * Re-reads the snapshot in the local explorer. Refresh happens only when the
 * viewer asks; reports opened from disk never show this control.
 */
export function RefreshControl({ explore }: { explore: ExploreConfig }) {
  const [state, setState] = useState<State>({ status: "idle" });

  const refresh = async () => {
    setState({ status: "refreshing" });
    try {
      const response = await fetch(explore.refresh, {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Tfviz-Token": explore.token },
      });
      if (response.ok) {
        window.location.reload();
        return;
      }
      const text = (await response.text()).trim().slice(0, 300);
      setState({ status: "failed", message: text || "The snapshot could not be read again." });
    } catch {
      setState({ status: "failed", message: "The explorer is no longer running. Start tfviz explore again." });
    }
  };

  return (
    <div className="refresh-control">
      <span className="mode-pill">Local explorer</span>
      <button type="button" className="secondary-button" onClick={refresh} disabled={state.status === "refreshing"}>
        {state.status === "refreshing" ? "Refreshing…" : "Refresh"}
      </button>
      {state.status === "failed" && (
        <span className="refresh-control__error" role="alert">
          {state.message}
        </span>
      )}
    </div>
  );
}
