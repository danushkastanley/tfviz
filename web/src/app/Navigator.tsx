import { useMemo, useRef, useState, type KeyboardEvent } from "react";
import type { ReportIndex } from "../report/view";
import { DEFAULT_FILTER, familiesIn, filterResources, type ChangeFilter, type Family, type ResourceFilter } from "../report/filter";
import { ACTION_ORDER, presentChange, type Action } from "../theme/changes";
import { FamilyIcon } from "../icons/FamilyIcon";

interface Props {
  index: ReportIndex;
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}

const FAMILY_LABEL: Record<Family, string> = {
  network: "Networking",
  security: "Security groups",
  compute: "Compute",
  load_balancing: "Load balancing",
  database: "Databases and caches",
  streaming: "Streaming",
  secrets: "Secrets",
  encryption: "Encryption",
  observability: "Observability",
  storage: "Storage",
  messaging: "Messaging",
  configuration: "Configuration",
  other: "Other",
};

/**
 * Keyboard-accessible list of every resource, grouped by change. The diagram
 * is never the only way to review a change.
 */
export function Navigator({ index, selectedId, onSelect }: Props) {
  const [filter, setFilter] = useState<ResourceFilter>(DEFAULT_FILTER);
  const listRef = useRef<HTMLUListElement>(null);
  const families = useMemo(() => familiesIn(index.report.resources), [index]);
  const update = (patch: Partial<ResourceFilter>) => setFilter((prev) => ({ ...prev, ...patch }));

  const sections = useMemo(() => {
    const visible = filterResources(index.report.resources, filter);
    return ACTION_ORDER.map((action) => ({ action, items: visible.filter((r) => r.change.action === action) })).filter(
      (s) => s.items.length > 0,
    );
  }, [index, filter]);

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
        <input type="search" placeholder="Search resources" value={filter.query} onChange={(e) => update({ query: e.target.value })} />
      </label>
      <div className="filter-row">
        <label>
          <span>Show</span>
          <select value={filter.change} onChange={(e) => update({ change: e.target.value as ChangeFilter })}>
            <option value="changes">Changes only</option>
            <option value="all">All resources</option>
            {ACTION_ORDER.filter((a) => a !== "no_op" && index.report.resources.some((r) => r.change.action === a)).map((a) => (
              <option key={a} value={a}>
                {presentChange(a).label}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>Domain</span>
          <select value={filter.family} onChange={(e) => update({ family: e.target.value as Family | "all" })}>
            <option value="all">All domains</option>
            {families.map((f) => (
              <option key={f} value={f}>
                {FAMILY_LABEL[f]}
              </option>
            ))}
          </select>
        </label>
      </div>
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
      {sections.length === 0 && <p className="inspector-empty">No resources match these filters.</p>}
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
