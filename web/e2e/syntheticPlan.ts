// Synthetic `terraform show -json` plans for end-to-end tests. They pass
// through the real reader, projection, graph and renderer via the CLI.

type Change = { actions: string[]; before: unknown; after: unknown };
interface ResourceChange {
  address: string;
  mode: "managed";
  type: string;
  name: string;
  index?: string | number;
  provider_name: string;
  change: Change & { after_unknown: object; before_sensitive: object; after_sensitive: object };
}

const PROVIDER = "registry.terraform.io/hashicorp/aws";
const ACCOUNT = "111122223333";
const arn = (service: string, resource: string) => `arn:aws:${service}:eu-west-1:${ACCOUNT}:${resource}`;

function rc(type: string, name: string, values: Record<string, unknown>, actions: string[] = ["no-op"], key?: string): ResourceChange {
  const address = key === undefined ? `${type}.${name}` : `${type}.${name}[${JSON.stringify(key)}]`;
  const before = actions[0] === "create" ? null : values;
  const after = actions.length === 1 && actions[0] === "delete" ? null : values;
  return {
    address,
    mode: "managed",
    type,
    name,
    ...(key === undefined ? {} : { index: key }),
    provider_name: PROVIDER,
    change: { actions, before, after, after_unknown: {}, before_sensitive: {}, after_sensitive: {} },
  };
}

function plan(resources: ResourceChange[]) {
  return {
    format_version: "1.2",
    terraform_version: "1.16.4",
    timestamp: "2026-10-03T09:00:00Z",
    complete: true,
    errored: false,
    resource_changes: resources,
  };
}

function network(azs: string[]): { resources: ResourceChange[]; subnets: string[] } {
  const resources = [rc("aws_vpc", "main", { id: "vpc-1", arn: arn("ec2", "vpc/vpc-1"), cidr_block: "10.0.0.0/16", tags: { Name: "synthetic" } })];
  const subnets: string[] = [];
  azs.forEach((az, i) => {
    const id = `subnet-${i}`;
    subnets.push(id);
    resources.push(rc("aws_subnet", "s", { id, arn: arn("ec2", `subnet/${id}`), vpc_id: "vpc-1", availability_zone: `eu-west-1${az}`, cidr_block: `10.0.${i}.0/24` }, ["no-op"], `${i}`));
  });
  return { resources, subnets };
}

/** Text that would inject markup or script if ever rendered as HTML. */
export const HOSTILE_TEXT = '</script><script>window.__pwned=1</script><img src=x onerror="window.__pwned=2"> \u2028 javascript:alert(1)';

/** A small plan whose names and keys are hostile. */
export function hostilePlan() {
  const { resources } = network(["a", "b"]);
  resources.push(
    rc("aws_security_group", "web", { id: "sg-1", arn: arn("ec2", "security-group/sg-1"), vpc_id: "vpc-1", name: HOSTILE_TEXT, tags: { Name: HOSTILE_TEXT } }, ["update"]),
    rc("aws_lb", "x", { arn: arn("elasticloadbalancing", "loadbalancer/app/x/1"), name: HOSTILE_TEXT, subnets: ["subnet-0", "subnet-1"], security_groups: ["sg-1"] }, ["create"], HOSTILE_TEXT),
  );
  return plan(resources);
}

/**
 * A plan with `count` resources and roughly three relationships each, for the
 * plan's performance target (500 resources, about 1,500 relationships).
 */
export function largePlan(count = 500) {
  const { resources, subnets } = network(["a", "b", "c", "a", "b", "c"]);
  const groups: string[] = [];
  for (let i = 0; i < 60; i++) {
    groups.push(`sg-${i}`);
    resources.push(rc("aws_security_group", "sg", { id: `sg-${i}`, arn: arn("ec2", `security-group/sg-${i}`), vpc_id: "vpc-1", name: `sg-${i}` }, ["no-op"], `${i}`));
  }
  const actions = [["no-op"], ["no-op"], ["update"], ["no-op"], ["create"], ["no-op"], ["delete", "create"], ["no-op"], ["delete"], ["no-op"]];
  const pick = <T,>(list: readonly T[], i: number) => list[i % list.length] as T;
  for (let i = 0; resources.length < count; i++) {
    const act = pick(actions, i);
    const sg = (k: number) => pick(groups, i * 7 + k);
    const subnet = (k: number) => pick(subnets, i + k);
    switch (i % 5) {
      case 0:
        resources.push(rc("aws_lb", "lb", { arn: arn("elasticloadbalancing", `loadbalancer/app/lb-${i}/1`), name: `lb-${i}`, subnets: [subnet(0), subnet(1), subnet(2)], security_groups: [sg(0)] }, act, `${i}`));
        break;
      case 1:
        resources.push(rc("aws_db_instance", "db", { id: `db-${i}`, arn: arn("rds", `db:db-${i}`), identifier: `db-${i}`, engine: "postgres", vpc_security_group_ids: [sg(1), sg(2), sg(3)] }, act, `${i}`));
        break;
      case 2:
        resources.push(rc("aws_nat_gateway", "nat", { id: `nat-${i}`, subnet_id: subnet(0) }, act, `${i}`));
        break;
      case 3:
        resources.push(rc("aws_vpc_security_group_ingress_rule", "rule", { id: `sgr-${i}`, security_group_id: sg(0), referenced_security_group_id: sg(4), ip_protocol: "tcp", from_port: 443, to_port: 443 }, act, `${i}`));
        break;
      default:
        resources.push(rc("aws_msk_cluster", "kafka", { arn: arn("kafka", `cluster/k-${i}/1`), cluster_name: `k-${i}`, broker_node_group_info: [{ client_subnets: [subnet(0), subnet(1), subnet(2)], security_groups: [sg(5)] }] }, act, `${i}`));
    }
  }
  return plan(resources);
}
