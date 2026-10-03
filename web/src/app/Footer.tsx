import type { Report } from "../report/schema.gen";
import { presentChange, type Action } from "../theme/changes";

interface Props {
  report: Report;
  reducedMotion: boolean;
  onReducedMotionChange: (value: boolean) => void;
}

const LEGEND: readonly Action[] = ["create", "update", "replace", "delete", "read"];

function completenessText(report: Report): string | undefined {
  switch (report.completeness.status) {
    case "incomplete":
      return "The producer reported this plan as incomplete.";
    case "not_reported":
      return "The producer did not report whether this plan is complete.";
    default:
      return undefined;
  }
}

export function Footer({ report, reducedMotion, onReducedMotionChange }: Props) {
  const { coverage } = report;
  const completeness = completenessText(report);
  return (
    <footer className="app-footer">
      <ul className="legend" aria-label="Legend">
        {LEGEND.map((action) => {
          const change = presentChange(action);
          return (
            <li key={action} className={`tone-${change.tone}`}>
              <span className="legend__swatch" aria-hidden="true">
                {change.symbol}
              </span>
              {change.label}
            </li>
          );
        })}
        <li className="legend__trace">Lines show recorded relationships, not traffic</li>
      </ul>
      <p className="coverage">
        {coverage.resources_supported} of {coverage.resources_total} resources fully supported
        {coverage.resources_generic > 0 && ` · ${coverage.resources_generic} shown with limited detail`}
        {report.unresolved.length > 0 && ` · ${report.unresolved.length} unresolved reference${report.unresolved.length === 1 ? "" : "s"}`}
        {completeness && ` · ${completeness}`}
      </p>
      <label className="toggle-row">
        <input type="checkbox" checked={reducedMotion} onChange={(e) => onReducedMotionChange(e.target.checked)} />
        Reduce motion
      </label>
    </footer>
  );
}
