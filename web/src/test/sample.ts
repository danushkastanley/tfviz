import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { parseReport } from "../report/load";
import type { Report } from "../report/schema.gen";

const SAMPLE = fileURLToPath(new URL("../../../testdata/reports/aws-review.sample.json", import.meta.url));

/** The hand-authored sample report shared by unit tests. */
export function loadSample(): Report {
  const result = parseReport(readFileSync(SAMPLE, "utf8"));
  if (!result.ok) throw new Error(result.message);
  return result.report;
}
