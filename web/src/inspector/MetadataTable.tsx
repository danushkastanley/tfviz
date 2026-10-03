import type { FieldValue, MetadataField } from "../report/schema.gen";
import { describeStatus } from "./labels";

function Value({ value, field }: { value: FieldValue; field: MetadataField }) {
  return <span className={`field-value status-${value.status}`}>{describeStatus(value, field)}</span>;
}

function FieldRow({ field }: { field: MetadataField }) {
  const { before, after, change_status: change } = field;
  const showsBoth = before !== undefined && after !== undefined && change !== undefined && change !== "unchanged";
  // Sensitive values have no payload on either side; one line says it all.
  const sensitiveOnly = before?.status === "sensitive" && after?.status === "sensitive";
  return (
    <div className={`field-row${change && change !== "unchanged" ? ` is-${change}` : ""}`}>
      <dt>{field.label}</dt>
      <dd>
        {showsBoth && !sensitiveOnly ? (
          <span className="field-diff">
            <Value value={before} field={field} />
            <span className="field-diff__arrow" aria-label="changes to">
              →
            </span>
            <Value value={after} field={field} />
          </span>
        ) : (
          (after ?? before) && <Value value={(after ?? before) as FieldValue} field={field} />
        )}
        {change === "removed" && <span className="field-note">Removed</span>}
      </dd>
    </div>
  );
}

export function MetadataTable({ fields }: { fields: readonly MetadataField[] }) {
  if (fields.length === 0) {
    return <p className="inspector-empty">No approved details are recorded for this resource type yet.</p>;
  }
  const changed = fields.filter((f) => f.change_status && f.change_status !== "unchanged");
  const unchanged = fields.filter((f) => !f.change_status || f.change_status === "unchanged");
  return (
    <>
      {changed.length > 0 && (
        <section aria-labelledby="inspector-changed">
          <h3 id="inspector-changed">Changes</h3>
          <dl className="field-list">
            {changed.map((f) => (
              <FieldRow key={f.key} field={f} />
            ))}
          </dl>
        </section>
      )}
      {unchanged.length > 0 && (
        <section aria-labelledby="inspector-details">
          <h3 id="inspector-details">Details</h3>
          <dl className="field-list">
            {unchanged.map((f) => (
              <FieldRow key={f.key} field={f} />
            ))}
          </dl>
        </section>
      )}
    </>
  );
}
