import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { parseReport } from "../report/load";
import type { Report } from "../report/schema.gen";

const SAMPLE = fileURLToPath(new URL("../../../testdata/golden/terraform-1.16-plan.json", import.meta.url));

/** The engine's golden report for the Terraform plan fixture. */
export function loadSample(): Report {
  const result = parseReport(readFileSync(SAMPLE, "utf8"));
  if (!result.ok) throw new Error(result.message);
  return result.report;
}

/** Looks up a resource ID by Terraform address. */
export function idOf(report: Report, address: string): string {
  const resource = report.resources.find((r) => r.address === address);
  if (!resource) throw new Error(`no resource ${address}`);
  return resource.id;
}

/** Looks up an architecture group ID by label prefix. */
export function groupId(report: Report, labelPrefix: string): string {
  const group = report.groups.find((g) => g.view === "architecture" && g.label.startsWith(labelPrefix));
  if (!group) throw new Error(`no group ${labelPrefix}`);
  return group.id;
}
