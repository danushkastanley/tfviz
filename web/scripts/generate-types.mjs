// Generates src/report/schema.gen.ts from the report JSON Schema.
// With --check, fails if the committed file is out of date (used in CI).
import { readFile, writeFile } from "node:fs/promises";
import { compile } from "json-schema-to-typescript";

const schemaPath = new URL("../../schema/report.v1.schema.json", import.meta.url);
const outPath = new URL("../src/report/schema.gen.ts", import.meta.url);

const schema = JSON.parse(await readFile(schemaPath, "utf8"));
const generated = await compile(schema, "Report", {
  bannerComment:
    "/* Generated from schema/report.v1.schema.json by scripts/generate-types.mjs. Do not edit. */",
  additionalProperties: false,
  unreachableDefinitions: true,
  style: { semi: true, singleQuote: false, printWidth: 100 },
});

if (process.argv.includes("--check")) {
  const current = await readFile(outPath, "utf8").catch(() => "");
  if (current !== generated) {
    console.error("src/report/schema.gen.ts is out of date. Run: pnpm schema:types");
    process.exit(1);
  }
  console.log("Schema types are up to date.");
} else {
  await writeFile(outPath, generated);
  console.log("Wrote src/report/schema.gen.ts");
}
