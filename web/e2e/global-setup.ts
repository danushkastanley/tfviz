import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { Report } from "../src/report/schema.gen";
import { largeReport } from "./largeReport";
import { HOSTILE_TEXT, REPORTS } from "./paths";

const repoRoot = fileURLToPath(new URL("../..", import.meta.url));
const sample = fileURLToPath(new URL("../../testdata/reports/aws-review.sample.json", import.meta.url));

/** Renders every e2e report through the real Go exporter. */
export default function globalSetup() {
  mkdirSync(REPORTS.dir, { recursive: true });
  const base = JSON.parse(readFileSync(sample, "utf8")) as Report;

  const hostile: Report = structuredClone(base);
  hostile.title = HOSTILE_TEXT;
  const first = hostile.resources.find((r) => r.id === "r36");
  if (!first) throw new Error("sample MSK resource missing");
  first.label = HOSTILE_TEXT;

  const inputs: [string, Report][] = [
    [REPORTS.sample, base],
    [REPORTS.hostile, hostile],
    [REPORTS.large, largeReport(base)],
  ];
  for (const [html, report] of inputs) {
    const json = html.replace(/\.html$/, ".json");
    writeFileSync(json, JSON.stringify(report));
    execFileSync("go", ["run", "./cmd/tfviz-spike", "-in", json, "-out", html, "-force"], { cwd: repoRoot, stdio: "inherit" });
  }
}
