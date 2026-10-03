import type { Relationship, Resource } from "../report/schema.gen";
import type { ReportIndex } from "../report/view";
import { EVIDENCE_LABEL, PRESENCE_LABEL, RELATIONSHIP_LABEL } from "./labels";

interface Props {
  index: ReportIndex;
  resource: Resource;
  onSelect: (id: string) => void;
}

/** Recorded associations with the evidence behind each. Not observed traffic. */
export function RelationshipList({ index, resource, onSelect }: Props) {
  const rels = index.relationshipsOf.get(resource.id) ?? [];
  const unresolved = index.report.unresolved.filter((u) => u.source === resource.id);
  if (rels.length === 0 && unresolved.length === 0) {
    return <p className="inspector-empty">No recorded relationships.</p>;
  }
  return (
    <ul className="relationship-list">
      {rels.map((rel) => (
        <RelationshipItem key={rel.id} rel={rel} index={index} self={resource.id} onSelect={onSelect} />
      ))}
      {unresolved.map((u) => (
        <li key={`${u.type}-${u.field}`} className="relationship-item is-unresolved">
          <span className="relationship-item__type">{RELATIONSHIP_LABEL[u.type]}</span>
          <span className="relationship-item__target">Not resolved: {UNRESOLVED_REASON[u.reason]}</span>
          <span className="relationship-item__evidence">
            Field <code>{u.field}</code>
          </span>
        </li>
      ))}
    </ul>
  );
}

const UNRESOLVED_REASON = {
  unknown_until_apply: "known after apply",
  external: "outside this report",
  ambiguous: "more than one possible match",
  sensitive: "withheld because the reference is sensitive",
} as const;

function RelationshipItem({ rel, index, self, onSelect }: { rel: Relationship; index: ReportIndex; self: string; onSelect: (id: string) => void }) {
  const otherId = rel.source === self ? rel.target : rel.target === self ? rel.source : undefined;
  const ends = otherId ? [otherId] : [rel.source, rel.target];
  const via = rel.evidence.via && rel.evidence.via !== self ? index.resources.get(rel.evidence.via) : undefined;
  return (
    <li className={`relationship-item presence-${rel.presence}`}>
      <span className="relationship-item__type">
        {RELATIONSHIP_LABEL[rel.type]}
        {rel.source !== self && otherId && <span className="relationship-item__direction"> (incoming)</span>}
      </span>
      <span className="relationship-item__target">
        {ends.map((id) => {
          const r = index.resources.get(id);
          return r ? (
            <button key={id} type="button" className="link-button" onClick={() => onSelect(id)}>
              {r.label}
            </button>
          ) : null;
        })}
      </span>
      <span className="relationship-item__evidence">
        {EVIDENCE_LABEL[rel.evidence.kind]} · <code>{rel.evidence.field}</code>
        {via && (
          <>
            {" via "}
            <button type="button" className="link-button" onClick={() => onSelect(via.id)}>
              {via.label}
            </button>
          </>
        )}
      </span>
      {rel.presence !== "both" && <span className="relationship-item__presence">{PRESENCE_LABEL[rel.presence]}</span>}
    </li>
  );
}
