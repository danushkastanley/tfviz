// Regenerates the website's demo reports and screenshots with the real CLI.
//
//   node scripts/site-assets.mjs --icons <AWS icon pack folder>
//
// Demo reports (site/static/reports) use tfviz's own symbols, so the site
// never redistributes provider icons. Screenshots are diagrams made with
// the official icons, which AWS permits; the pack itself is never copied.
// Run `make site` afterwards to publish the result into site/public.
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";
import { chromium } from "@playwright/test";
import { largePlan } from "../e2e/syntheticPlan.ts";

const { values } = parseArgs({ options: { icons: { type: "string" } } });
if (!values.icons) throw new Error("--icons <folder> is required: the unzipped AWS Architecture Icons pack");

const repo = fileURLToPath(new URL("../..", import.meta.url));
const work = fileURLToPath(new URL("../test-results/site-assets/", import.meta.url));
const reportsOut = `${repo}site/static/reports/`;
const imagesOut = `${repo}site/static/images/`;
for (const dir of [work, reportsOut, imagesOut]) mkdirSync(dir, { recursive: true });

const tfviz = `${work}tfviz`;
execFileSync("go", ["build", "-o", tfviz, "./cmd/tfviz"], { cwd: repo, stdio: "inherit" });
const fixture = (p) => `${repo}testdata/producer/${p}`;
const largeJSON = `${work}large.plan.json`;
writeFileSync(largeJSON, JSON.stringify(largePlan()));

const reports = {
  "plan-review": ["plan", fixture("terraform-1.16/plan.json"), "--title", "aws-review · proposed change"],
  "safe-share": ["plan", fixture("terraform-1.16/plan.json"), "--safe-share"],
  "platform-plan": ["plan", fixture("terraform-1.16-platform/plan.json"), "--title", "aws-platform · proposed change"],
  "platform-state": ["state", fixture("terraform-1.16-platform/state.json"), "--title", "aws-platform · recorded state"],
  "large-plan": ["plan", largeJSON, "--title", "500-resource stack · proposed change"],
};
const published = new Set(["plan-review", "safe-share", "platform-state", "large-plan"]);
const render = (name, [command, input, ...extra], out, more = []) =>
  execFileSync(tfviz, [command, "--input", input, "--output", out, "--force", ...extra, ...more], { stdio: "inherit" });
for (const [name, args] of Object.entries(reports)) {
  if (published.has(name)) render(name, args, `${reportsOut}${name}.html`);
  render(name, args, `${work}${name}.html`, ["--icons", values.icons]);
}

const browser = await chromium.launch();
const encoder = await browser.newPage();

/** Converts a PNG screenshot to WebP in the browser, so no image tools are needed. */
async function webp(png, path, quality = 0.86) {
  const dataURL = await encoder.evaluate(async ({ b64, quality }) => {
    const img = new Image();
    img.src = `data:image/png;base64,${b64}`;
    await img.decode();
    const canvas = Object.assign(document.createElement("canvas"), { width: img.naturalWidth, height: img.naturalHeight });
    canvas.getContext("2d").drawImage(img, 0, 0);
    return canvas.toDataURL("image/webp", quality);
  }, { b64: png.toString("base64"), quality });
  writeFileSync(path, Buffer.from(dataURL.split(",")[1], "base64"));
}

const select = (label) => (p) => p.getByRole("button", { name: new RegExp(label) }).first().click();
const fit = async (p) => { await p.waitForTimeout(400); await p.locator(".react-flow__controls-fitview").click(); };
const then = (...steps) => async (p) => { for (const step of steps) await step(p); };

async function shot(out, report, scheme, act, { clip = false } = {}) {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 2, colorScheme: scheme });
  const problems = [];
  page.on("console", (m) => ["error", "warning"].includes(m.type()) && problems.push(m.text()));
  await page.context().route(/^(?!file:)/, (r) => r.abort());
  await page.goto(`file://${work}${report}.html`);
  await page.locator(".react-flow__node").first().waitFor();
  if (act) await act(page);
  await page.waitForTimeout(800);
  let area;
  if (clip) {
    const box = await page.locator(".react-flow__node-group").evaluateAll((els) => {
      const r = els.map((e) => e.getBoundingClientRect());
      const [l, t] = [Math.min(...r.map((x) => x.left)), Math.min(...r.map((x) => x.top))];
      return { x: l - 16, y: t - 16, width: Math.max(...r.map((x) => x.right)) - l + 32, height: Math.max(...r.map((x) => x.bottom)) - t + 32 };
    });
    area = box;
  }
  if (problems.length) throw new Error(`${out}: ${problems.join("; ")}`);
  await webp(await page.screenshot({ clip: area }), `${imagesOut}${out}.webp`);
  console.log(`images/${out}.webp`);
  await page.close();
}

await shot("hero-light", "plan-review", "light", then(select("review-events"), fit));
await shot("secrets-dark", "plan-review", "dark", select("review-orders"));
await shot("diagram-light", "plan-review", "light", null, { clip: true });
await shot("trace-light", "plan-review", "light", select("review-events"));
await shot("state-light", "platform-state", "light", select("platform-stream"));
await shot("modules-light", "plan-review", "light", (p) => p.getByRole("radio", { name: "Modules" }).click());
await shot("safe-share-light", "safe-share", "light", (p) => p.locator("button[data-resource]").nth(3).click());
await shot("platform-dark", "platform-plan", "dark", null, { clip: true });
await shot("large-dark", "large-plan", "dark", then((p) => p.getByRole("button", { name: "Expand VPC synthetic" }).click(), fit));
await openGraph();
await browser.close();

/** Renders the 1200×630 social preview from the hero screenshot and the site's font. */
async function openGraph() {
  const page = await browser.newPage({ viewport: { width: 1200, height: 630 } });
  const html = `<!doctype html><meta charset="utf-8"><style>
    @font-face { font-family: Inter; src: url("${repo}site/static/fonts/inter-latin.woff2"); font-weight: 400 700; }
    body { margin: 0; width: 1200px; height: 630px; overflow: hidden; font-family: Inter, sans-serif; color: #0c0d12;
      background: radial-gradient(120% 90% at 0% 0%, #e7ebff 0%, transparent 55%), radial-gradient(110% 90% at 20% 100%, #ffd9cb 0%, transparent 60%),
        radial-gradient(80% 70% at 90% 100%, #ffe8de 0%, transparent 60%), #f7f8fd; }
    .text { position: absolute; left: 72px; top: 76px; width: 470px; }
    .brand { display: flex; gap: 12px; align-items: center; font-weight: 700; font-size: 26px; letter-spacing: -0.02em; }
    h1 { margin: 44px 0 20px; font-size: 52px; line-height: 1.05; letter-spacing: -0.035em; font-weight: 650; }
    p { margin: 0; color: #646a7a; font-size: 22px; line-height: 1.45; }
    img { position: absolute; left: 600px; top: 80px; width: 820px; border-radius: 14px; box-shadow: 0 24px 60px rgba(31, 45, 90, 0.18); }
  </style>
  <div class="text"><div class="brand"><svg viewBox="0 0 32 32" width="36" height="36"><rect width="32" height="32" rx="8" fill="#0c0d12"/><path d="M9 11h14M9 16h9M9 21h11" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/></svg>tfviz</div>
  <h1>See your Terraform plan before you apply it</h1><p>Plans and state as calm, private architecture diagrams. One offline HTML file.</p></div>
  <img src="${imagesOut}hero-light.webp" alt="">`;
  const file = `${work}og.html`;
  writeFileSync(file, html);
  await page.goto(`file://${file}`);
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({ path: `${imagesOut}og.png` });
  console.log("images/og.png");
  await page.close();
}
