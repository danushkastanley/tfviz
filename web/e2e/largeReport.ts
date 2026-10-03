import type { Group, MetadataField, Relationship, Report, Resource } from "../src/report/schema.gen";

type Family = Resource["family"];
type Action = Resource["change"]["action"];

const AZS = ["a", "b", "c"] as const;
const FAMILIES: readonly Family[] = ["compute", "database", "security", "load_balancing", "streaming", "secrets", "observability", "messaging"];
const REGIONAL: ReadonlySet<Family> = new Set(["secrets", "observability", "messaging"]);
const ACTIONS: readonly Action[] = ["no_op", "no_op", "no_op", "no_op", "update", "create", "no_op", "replace", "no_op", "delete"];
const REL_TYPES: readonly Relationship["type"][] = ["service_reference", "security_group_attachment", "encryption_key", "log_destination"];

/** Deterministic pseudo-random sequence so the fixture is stable between runs. */
function rng(seed: number) {
  let s = seed;
  return () => {
    s = (s * 1664525 + 1013904223) % 4294967296;
    return s / 4294967296;
  };
}

const known = (key: string, label: string, value: string | number): MetadataField => ({
  key,
  label,
  after: { status: "known", value },
  change_status: "unchanged",
});

/**
 * A synthetic stack of `resourceCount` resources and roughly three
 * relationships per resource, for the performance budget in the plan
 * (500 resources and about 1,500 relationships).
 */
export function largeReport(base: Report, resourceCount = 500): Report {
  const random = rng(42);
  const groups: Group[] = [
    { id: "g1", view: "architecture", kind: "account", label: "Account 111122223333", placement: "known" },
    { id: "g2", view: "architecture", kind: "region", label: "eu-west-1", parent: "g1", placement: "known" },
    { id: "g3", view: "architecture", kind: "vpc", label: "large-platform", parent: "g2", placement: "known", resource: "r1" },
    { id: "g10", view: "architecture", kind: "regional_services", label: "Regional services", parent: "g2", placement: "known" },
    { id: "m1", view: "modules", kind: "module", label: "Root module", placement: "known" },
  ];
  const resource = (id: string, type: string, family: Family, group: string, action: Action, metadata: MetadataField[]): Resource => ({
    id,
    address: `${type}.${id}`,
    label: `${family}-${id}`,
    type,
    family,
    role: "entity",
    mode: "managed",
    module: "",
    provider: "hashicorp/aws",
    support: "supported",
    change: { action, producer_actions: action === "replace" ? ["delete", "create"] : [action === "no_op" ? "no-op" : action] },
    groups: { architecture: group, modules: "m1" },
    metadata,
  });

  const resources: Resource[] = [resource("r1", "aws_vpc", "network", "g2", "no_op", [known("cidr_block", "IPv4 CIDR", "10.0.0.0/16")])];
  const subnetIds: string[] = [];
  let n = 2;
  for (const tier of ["public", "private"] as const) {
    for (const az of AZS) {
      const id = `r${n++}`;
      const gid = `g${3 + n}`;
      subnetIds.push(id);
      groups.push({ id: gid, view: "architecture", kind: "subnet", label: `${tier}-${az}`, parent: "g3", placement: "known", resource: id, classification: tier });
      resources.push(resource(id, "aws_subnet", "network", "g3", "no_op", [known("availability_zone", "Availability zone", `eu-west-1${az}`)]));
    }
  }
  const subnetGroups = groups.filter((g) => g.kind === "subnet").map((g) => g.id);

  const entities: string[] = [];
  while (resources.length < resourceCount) {
    const id = `r${n++}`;
    const family = FAMILIES[resources.length % FAMILIES.length] as Family;
    const group = REGIONAL.has(family) ? "g10" : family === "compute" ? (subnetGroups[resources.length % subnetGroups.length] as string) : "g3";
    const action = ACTIONS[Math.floor(random() * ACTIONS.length)] as Action;
    resources.push(resource(id, `aws_${family}_thing`, family, group, action, [known("name", "Name", `${family}-${id}`), known("size", "Size", Math.floor(random() * 100))]));
    entities.push(id);
  }

  const relationships: Relationship[] = [];
  for (const source of entities) {
    for (let k = 0; k < 3; k++) {
      const target = entities[Math.floor(random() * entities.length)] as string;
      if (target === source) continue;
      relationships.push({
        id: `e${relationships.length + 1}`,
        source,
        target,
        type: REL_TYPES[k % REL_TYPES.length] as Relationship["type"],
        presence: "both",
        evidence: { kind: "attribute", field: "reference" },
      });
    }
  }

  const summary = { create: 0, update: 0, delete: 0, replace: 0, read: 0, no_op: 0, unsupported: 0 };
  for (const r of resources) summary[r.change.action] += 1;
  return {
    ...base,
    title: `Synthetic stack · ${resources.length} resources`,
    summary,
    coverage: { resources_total: resources.length, resources_supported: resources.length, resources_generic: 0 },
    resources,
    relationships,
    unresolved: [],
    groups,
    warnings: [],
  };
}
