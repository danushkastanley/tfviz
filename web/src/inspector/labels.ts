import type { DisplayValue, FieldValue, MetadataField, Relationship } from "../report/schema.gen";

export const RELATIONSHIP_LABEL: Record<Relationship["type"], string> = {
  subnet_membership: "Subnet membership",
  security_group_attachment: "Security group attachment",
  security_group_rule: "Security group rule",
  routing: "Route",
  route_association: "Route table association",
  service_reference: "Configured reference",
  secret_association: "Secret association",
  encryption_key: "Encryption key",
  log_destination: "Log destination",
};

export const EVIDENCE_LABEL: Record<Relationship["evidence"]["kind"], string> = {
  attribute: "Recorded attribute",
  configuration: "Configuration reference (value known after apply)",
  association_resource: "Association resource",
};

export const PRESENCE_LABEL: Record<Relationship["presence"], string> = {
  before: "Before the change only",
  after: "After the change only",
  both: "Before and after",
};

export function formatDisplayValue(value: DisplayValue): string {
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (Array.isArray(value)) return value.length === 0 ? "None" : value.join("\n");
  return String(value);
}

/** Text for a value that carries no payload. Never implies a masked secret is present. */
export function describeStatus(value: FieldValue, field: MetadataField): string {
  switch (value.status) {
    case "known":
      return formatDisplayValue(value.value);
    case "sensitive":
      if (field.change_status === "changed") return "Sensitive value changed";
      if (field.change_status === "unknown") return "Sensitive value; comparison unavailable";
      if (field.change_status === "added") return "Sensitive value set";
      return "Sensitive value";
    case "unknown":
      return "Known after apply";
    case "absent":
      return "Not set";
    case "omitted":
      return "Not included in reports";
  }
}

/** Plain-text form used for copying; payload-free statuses copy nothing. */
export function copyText(value: FieldValue | undefined): string | undefined {
  if (value?.status !== "known") return undefined;
  return formatDisplayValue(value.value);
}
