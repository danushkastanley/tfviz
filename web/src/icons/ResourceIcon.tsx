import { useReportIndex } from "../app/reportContext";
import type { Report, Resource } from "../report/schema.gen";
import { FamilyIcon } from "./FamilyIcon";

/** The data URI of the user's icon for a resource type, if the report has one. */
export function iconFor(report: Report, type: string): string | undefined {
  const id = report.icons?.by_type[type];
  return id === undefined ? undefined : report.icons?.images[id];
}

/**
 * Shows the official service icon when the report was generated with
 * --icons, and tfviz's own family symbol otherwise. Icons are decorative:
 * the label and type beside them carry the meaning.
 */
export function ResourceIcon({ resource, size = 16 }: { resource: Resource; size?: number }) {
  const { report } = useReportIndex();
  const src = iconFor(report, resource.type);
  if (src === undefined) return <FamilyIcon family={resource.family} size={size} />;
  return <img className="resource-icon" src={src} width={size} height={size} alt="" aria-hidden="true" draggable={false} />;
}
