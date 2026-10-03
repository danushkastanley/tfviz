import { useMemo, useRef, useState, type KeyboardEvent } from "react";
import type { Resource } from "../report/schema.gen";
import type { ReportIndex } from "../report/view";
import { ACTION_ORDER, presentChange, type Action } from "../theme/changes";
import { FamilyIcon } from "../icons/FamilyIcon";

interface Props {
  index: ReportIndex;
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}

function matches(resource: Resource, query: string): boolean {
  if (query === "") return true;
  const q = query.toLowerCase();
  return [resource.label, resource.address, resource.type].some((s) => s.toLowerCase().includes(q));
}

/**
 * Keyboard-accessible list of every resource, grouped by change. The diagram
 * is never the only way to review a change.
 */
export function Navigator({ index, selectedId, onSelect }: Props) {
  const [query, setQuery] = useState("");
  const [hideUnchanged, setHideUnchanged] = useState(true);
  const listRef = useRef<HTMLUListElement>(null);

  const sections = useMemo(() => {
    const visible = index.report.resources.filter(
      (r) => matches(r, query.trim()) && !(hideUnchanged && r.change.action === "no_op" && query.trim() === ""),
    );
    return ACTION_ORDER.map((action) => ({ action, items: visible.filter((r) => r.change.action === action) })).filter(
      (s) => s.items.length > 0,
    );
  }, [index, query, hideUnchanged]);

  const onKeyDown = (event: KeyboardEvent<HTMLUListElement>) => {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    const buttons = Array.from(listRef.current?.querySelectorAll<HTMLButtonElement>("button[data-resource]") ?? []);
    const current = buttons.indexOf(document.activeElement as HTMLButtonElement);
    const next = buttons[current + (event.key === "ArrowDown" ? 1 : -1)];
    if (next) {
      event.preventDefault();
      next.focus();
    }
  };

  const total = index.report.resources.length;
  return (
    <nav className="navigator" aria-label="Resources">
      <label className="search-field">
        <span className="visually-hidden">Search resources</span>
        <input type="search" placeholder="Search resources" value={query} onChange={(e) => setQuery(e.target.value)} />
      </label>
      <label className="toggle-row">
        <input type="checkbox" checked={!hideUnchanged} onChange={(e) => setHideUnchanged(!e.target.checked)} />
        Show unchanged resources
      </label>
      <ul className="resource-list" ref={listRef} onKeyDown={onKeyDown}>
        {sections.map(({ action, items }) => (
          <li key={action}>
            <h2 className="resource-list__heading">
              <SectionTitle action={action} count={items.length} />
            </h2>
            <ul>
              {items.map((r) => (
                <li key={r.id}>
                  <button
                    type="button"
                    data-resource={r.id}
                    className={`resource-list__item${r.id === selectedId ? " is-selected" : ""}`}
                    aria-current={r.id === selectedId ? "true" : undefined}
                    onClick={() => onSelect(r.id)}
                  >
                    <FamilyIcon family={r.family} size={14} />
                    <span className="resource-list__label">{r.label}</span>
                    <span className="resource-list__address">{r.address}</span>
                  </button>
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
      {sections.length === 0 && <p className="inspector-empty">No resources match “{query}”.</p>}
      <p className="navigator__count">{total} resources in this report</p>
    </nav>
  );
}

function SectionTitle({ action, count }: { action: Action; count: number }) {
  const change = presentChange(action);
  return (
    <span className={`tone-${change.tone}`}>
      {change.symbol && <span aria-hidden="true">{change.symbol} </span>}
      {change.label} <span className="resource-list__count">{count}</span>
    </span>
  );
}
