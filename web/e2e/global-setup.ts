import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { REPORTS } from "./paths";
import { HOSTILE_TEXT, hostilePlan, largePlan } from "./syntheticPlan";

const repoRoot = fileURLToPath(new URL("../..", import.meta.url));
const fixturePlan = fileURLToPath(new URL("../../testdata/producer/terraform-1.16/plan.json", import.meta.url));
const placeholderIcons = fileURLToPath(new URL("../../testdata/icons/aws", import.meta.url));

/** Renders every e2e report with the real tfviz CLI. */
export default function globalSetup() {
  mkdirSync(REPORTS.dir, { recursive: true });
  execFileSync("go", ["build", "-o", `${REPORTS.dir}tfviz`, "./cmd/tfviz"], { cwd: repoRoot, stdio: "inherit" });
  const tfviz = (input: string, output: string, ...extra: string[]) =>
    execFileSync(`${REPORTS.dir}tfviz`, ["plan", "--input", input, "--output", output, "--force", ...extra], { cwd: repoRoot, stdio: "inherit" });

  tfviz(fixturePlan, REPORTS.sample, "--title", "aws-review · proposed change");
  tfviz(fixturePlan, REPORTS.safeShare, "--safe-share", "--title", "aws-review · proposed change");
  tfviz(fixturePlan, REPORTS.icons, "--icons", placeholderIcons);
  for (const [html, doc, extra] of [
    [REPORTS.hostile, hostilePlan(), ["--title", HOSTILE_TEXT]],
    [REPORTS.large, largePlan(), []],
  ] as const) {
    const json = html.replace(/\.html$/, ".plan.json");
    writeFileSync(json, JSON.stringify(doc));
    tfviz(json, html, ...extra);
  }
}
