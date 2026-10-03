import { presentChange, type Action } from "../theme/changes";

/** Change marker that pairs colour with a symbol and a word. */
export function ChangeBadge({ action, compact = false }: { action: Action; compact?: boolean }) {
  const change = presentChange(action);
  if (change.tone === "none" && action === "no_op") return null;
  return (
    <span className={`change-badge tone-${change.tone}${compact ? " is-compact" : ""}`}>
      {change.symbol && (
        <span className="change-badge__symbol" aria-hidden="true">
          {change.symbol}
        </span>
      )}
      <span className="change-badge__label">{change.label}</span>
    </span>
  );
}
