// Writes dist/THIRD_PARTY_NOTICES.txt with the licence text of every
// production dependency bundled into reports. Fails on licences outside the
// allowlist or packages without a licence file, so a new dependency cannot
// enter reports unreviewed.
import { execFileSync } from "node:child_process";
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const ALLOWED = new Set(["MIT", "ISC", "BSD-2-Clause", "BSD-3-Clause", "Apache-2.0", "0BSD"]);

const listing = JSON.parse(execFileSync("pnpm", ["licenses", "list", "--prod", "--json"], { encoding: "utf8" }));
const entries = [];
for (const [licence, packages] of Object.entries(listing)) {
  for (const pkg of packages) {
    // Type declarations are never bundled.
    if (pkg.name.startsWith("@types/")) continue;
    if (!ALLOWED.has(licence)) {
      throw new Error(`${pkg.name} uses ${licence}, which is not on the licence allowlist.`);
    }
    for (const [i, version] of pkg.versions.entries()) {
      const dir = pkg.paths[i];
      const file = readdirSync(dir).find((f) => /^licen[cs]e(\.|$)/i.test(f));
      if (!file) throw new Error(`${pkg.name}@${version} has no licence file.`);
      entries.push({ name: pkg.name, version, licence, text: readFileSync(join(dir, file), "utf8").trim() });
    }
  }
}
entries.sort((a, b) => a.name.localeCompare(b.name) || a.version.localeCompare(b.version));

const header = "tfviz reports include the following third-party software.\n";
const body = entries.map((e) => `${"=".repeat(72)}\n${e.name} ${e.version} (${e.licence})\n${"=".repeat(72)}\n${e.text}\n`).join("\n");
writeFileSync(new URL("../dist/THIRD_PARTY_NOTICES.txt", import.meta.url), `${header}\n${body}`);
console.log(`Wrote notices for ${entries.length} packages.`);
