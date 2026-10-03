import type { Relationship, Report, Resource } from "./schema.gen";

/** Which side of a plan the canvas shows. State reports always use "after". */
export type ViewMode = "changes" | "before" | "after";

const ABSENT_BEFORE: ReadonlySet<Resource["change"]["action"]> = new Set(["create", "read"]);
const ABSENT_AFTER: ReadonlySet<Resource["change"]["action"]> = new Set(["delete", "forget"]);

export function resourceInView(resource: Resource, view: ViewMode): boolean {
  if (view === "before") return !ABSENT_BEFORE.has(resource.change.action);
  if (view === "after") return !ABSENT_AFTER.has(resource.change.action);
  return true;
}

export function relationshipInView(rel: Relationship, view: ViewMode): boolean {
  if (view === "before") return rel.presence !== "after";
  if (view === "after") return rel.presence !== "before";
  return true;
}

/** Fast lookups shared by the canvas, navigator and inspector. */
export interface ReportIndex {
  report: Report;
  resources: ReadonlyMap<string, Resource>;
  /** Relationships touching a resource as source, target or association. */
  relationshipsOf: ReadonlyMap<string, readonly Relationship[]>;
  /** Group id defined by a resource, such as a VPC or subnet. */
  groupOfResource: ReadonlyMap<string, string>;
}

export function indexReport(report: Report): ReportIndex {
  const resources = new Map(report.resources.map((r) => [r.id, r]));
  const relationshipsOf = new Map<string, Relationship[]>();
  const add = (id: string, rel: Relationship) => {
    const list = relationshipsOf.get(id);
    if (list) {
      if (!list.includes(rel)) list.push(rel);
    } else {
      relationshipsOf.set(id, [rel]);
    }
  };
  for (const rel of report.relationships) {
    add(rel.source, rel);
    add(rel.target, rel);
    if (rel.evidence.via) add(rel.evidence.via, rel);
  }
  const groupOfResource = new Map<string, string>();
  for (const group of report.groups) {
    if (group.resource) groupOfResource.set(group.resource, group.id);
  }
  return { report, resources, relationshipsOf, groupOfResource };
}
