import type { Report } from "../report/schema.gen";
import type { ViewMode } from "../report/view";
import type { Structure } from "../graph/canvasModel";
import { presentChange, type Action } from "../theme/changes";
import type { ThemePreference } from "./preferences";

interface Props {
  report: Report;
  view: ViewMode;
  onViewChange: (view: ViewMode) => void;
  structure: Structure;
  onStructureChange: (structure: Structure) => void;
  theme: ThemePreference;
  onThemeChange: (theme: ThemePreference) => void;
}

const SUMMARY_ACTIONS: readonly (Action & keyof Report["summary"])[] = ["create", "update", "replace", "delete", "forget", "read"];
const STRUCTURES: readonly { id: Structure; label: string }[] = [
  { id: "architecture", label: "Architecture" },
  { id: "modules", label: "Modules" },
];

const VIEWS: readonly { id: ViewMode; label: string }[] = [
  { id: "changes", label: "Changes" },
  { id: "before", label: "Before" },
  { id: "after", label: "After" },
];

const dateFormat = new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" });

function freshness(report: Report): string {
  const noun = report.mode === "plan" ? "Plan" : "State snapshot";
  if (report.source.timestamp_status !== "known" || !report.source.timestamp) {
    return `${noun} time not recorded`;
  }
  return `${noun} from ${dateFormat.format(new Date(report.source.timestamp))} UTC`;
}

export function Header({ report, view, onViewChange, structure, onStructureChange, theme, onThemeChange }: Props) {
  return (
    <header className="app-header">
      <div className="app-header__title">
        <h1>{report.title}</h1>
        <p>
          <span className="mode-pill">{report.mode === "plan" ? "Plan review" : "Recorded state"}</span>
          <span>{freshness(report)}</span>
          {report.disclosure === "safe_share" && <span className="mode-pill">Safe-share</span>}
        </p>
      </div>

      {report.mode === "plan" && (
        <ul className="change-summary" aria-label="Change summary">
          {SUMMARY_ACTIONS.filter((a) => report.summary[a] > 0).map((action) => {
            const change = presentChange(action);
            return (
              <li key={action} className={`summary-chip tone-${change.tone}`}>
                <span aria-hidden="true">{change.symbol}</span> {report.summary[action]} {change.label.toLowerCase()}
              </li>
            );
          })}
        </ul>
      )}

      <div className="app-header__controls">
        <div className="segmented" role="radiogroup" aria-label="Organise by">
          {STRUCTURES.map((s) => (
            <button key={s.id} type="button" role="radio" aria-checked={structure === s.id} onClick={() => onStructureChange(s.id)}>
              {s.label}
            </button>
          ))}
        </div>
        {report.mode === "plan" && (
          <div className="segmented" role="radiogroup" aria-label="Diagram view">
            {VIEWS.map((v) => (
              <button key={v.id} type="button" role="radio" aria-checked={view === v.id} onClick={() => onViewChange(v.id)}>
                {v.label}
              </button>
            ))}
          </div>
        )}
        <label className="select-control">
          <span className="visually-hidden">Theme</span>
          <select value={theme} onChange={(e) => onThemeChange(e.target.value as ThemePreference)}>
            <option value="system">System theme</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </label>
      </div>
    </header>
  );
}
