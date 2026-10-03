import { expect, test, type Page } from "@playwright/test";
import { HOSTILE_TEXT, REPORTS, fileUrl } from "./paths";

interface Watch {
  problems: string[];
  requests: string[];
}

/** Records console errors, page errors, CSP violations and every request. */
async function open(page: Page, path: string): Promise<Watch> {
  const watch: Watch = { problems: [], requests: [] };
  // Abort anything that is not the report file itself. Browser offline modes
  // are inconsistent for file:// (WebKit refuses the file too), and the
  // request log below is the evidence that nothing else was attempted.
  await page.context().route(/^(?!file:)/, (route) => route.abort());
  page.on("console", (msg) => {
    if (msg.type() === "error" || msg.type() === "warning") watch.problems.push(`${msg.type()}: ${msg.text()}`);
  });
  page.on("pageerror", (err) => watch.problems.push(`pageerror: ${err.message}`));
  page.on("request", (req) => watch.requests.push(req.url()));
  await page.addInitScript(() => {
    document.addEventListener("securitypolicyviolation", (e) => {
      console.error(`CSP violation: ${e.violatedDirective} ${e.blockedURI}`);
    });
  });
  await page.goto(fileUrl(path));
  await expect(page.locator(".react-flow__node").first()).toBeVisible();
  return watch;
}

function expectCleanAndOffline(watch: Watch, path: string) {
  expect(watch.problems).toEqual([]);
  expect(watch.requests.filter((url) => url !== fileUrl(path))).toEqual([]);
}

test("the sample report works from file:// with no network, errors or CSP violations", async ({ page }) => {
  const watch = await open(page, REPORTS.sample);
  await expect(page.locator("#static-summary")).toHaveCount(0);
  expect(await page.locator(".react-flow__node").count()).toBeGreaterThan(30);

  await page.getByRole("button", { name: /review-events/ }).first().click();
  await expect(page.getByRole("complementary", { name: /Inspector: review-events/ })).toBeVisible();
  await expect(page.getByText("kafka.m7g.xlarge")).toBeVisible();

  await page.getByRole("radio", { name: "Before" }).click();
  await expect(page.locator('.react-flow__node[data-id="r37"]')).toHaveCount(0);
  expectCleanAndOffline(watch, REPORTS.sample);
});

test("the inspector never shows a payload for sensitive values", async ({ page }) => {
  await open(page, REPORTS.sample);
  await page.getByRole("button", { name: /review-orders/ }).first().click();
  await expect(page.getByText("Sensitive value changed")).toBeVisible();
  await expect(page.getByText("Not included in reports").first()).toBeVisible();
});

test("hostile labels render as text and never execute", async ({ page }) => {
  const watch = await open(page, REPORTS.hostile);
  await expect(page.locator(".app-header h1")).toHaveText(HOSTILE_TEXT);
  expect(await page.evaluate(() => (window as unknown as { __pwned?: number }).__pwned)).toBeUndefined();
  expect(await page.locator("img").count()).toBe(0);
  expect(await page.title()).toBe(HOSTILE_TEXT);
  expectCleanAndOffline(watch, REPORTS.hostile);
});

test.describe("without JavaScript", () => {
  test.use({ javaScriptEnabled: false });

  test("the static summary still lists every change", async ({ page }) => {
    await page.goto(fileUrl(REPORTS.sample));
    const summary = page.locator("#static-summary");
    await expect(summary).toBeVisible();
    await expect(summary.getByText("module.streaming.aws_msk_cluster.events")).toBeVisible();
    await expect(summary.locator("tbody tr")).toHaveCount(13);
  });
});

test("a 500-resource report reaches a useful view within the budget", async ({ page }, info) => {
  const started = Date.now();
  const watch = await open(page, REPORTS.large);
  const firstView = Date.now() - started;
  const layoutMs = Number(await page.evaluate(() => document.documentElement.dataset.layoutMs));
  info.annotations.push({ type: "first-useful-view-ms", description: String(firstView) });
  info.annotations.push({ type: "layout-ms", description: String(layoutMs) });
  console.log(`[${info.project.name}] 500 resources: first useful view ${firstView} ms, layout ${layoutMs} ms`);
  expect(firstView).toBeLessThan(3000);
  expectCleanAndOffline(watch, REPORTS.large);
});

test("the report's CSP actively blocks injected inline script", async ({ page }) => {
  const watch = await open(page, REPORTS.sample);
  await page.evaluate(() => {
    const script = document.createElement("script");
    script.textContent = "window.__injected = 1";
    document.body.append(script);
  });
  expect(await page.evaluate(() => (window as unknown as { __injected?: number }).__injected)).toBeUndefined();
  await expect.poll(() => watch.problems.some((p) => p.includes("CSP violation: script-src"))).toBe(true);
});
