import { expect, test, type Page } from "@playwright/test";
import { REPORTS, fileUrl } from "./paths";
import { HOSTILE_TEXT } from "./syntheticPlan";

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

  const createdSecret = page.locator(".react-flow__node").filter({ hasText: "AmazonMSK_review_orders" });
  await expect(createdSecret).toHaveCount(1);
  await page.getByRole("radio", { name: "Before" }).click();
  await expect(createdSecret).toHaveCount(0);
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
  await expect(page.locator(".react-flow__node").filter({ hasText: "window.__pwned=1" }).first()).toBeVisible();
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

test("a 500-resource report opens collapsed, useful and within budget", async ({ page }, info) => {
  const started = Date.now();
  const watch = await open(page, REPORTS.large);
  const firstView = Date.now() - started;
  const layoutMs = Number(await page.evaluate(() => document.documentElement.dataset.layoutMs));
  info.annotations.push({ type: "first-useful-view-ms", description: String(firstView) });
  info.annotations.push({ type: "layout-ms", description: String(layoutMs) });
  console.log(`[${info.project.name}] 500 resources: first useful view ${firstView} ms, layout ${layoutMs} ms`);
  expect(firstView).toBeLessThan(3000);

  // Large groups start collapsed with a summary rather than hundreds of cards.
  await expect(page.getByText(/\d+ resources · \d+ changing/).first()).toBeVisible();
  expect(await page.locator(".react-flow__node .resource-card").count()).toBeLessThan(50);

  // Search finds a resource inside a collapsed group, expands its path and focuses it.
  await page.getByPlaceholder("Search resources").fill("lb-0");
  await page.getByRole("button", { name: /^lb-0\b/ }).first().click();
  const card = page.locator(".react-flow__node .resource-card").filter({ hasText: "lb-0" }).first();
  await expect(card).toBeVisible();
  await expect(page.getByRole("complementary", { name: /Inspector: lb-0/ })).toBeVisible();
  await expect.poll(() => page.locator(".react-flow__edge").count()).toBeGreaterThan(0);
  expectCleanAndOffline(watch, REPORTS.large);
});

test("groups collapse and expand from their header", async ({ page }) => {
  await open(page, REPORTS.sample);
  await page.getByRole("button", { name: "Collapse VPC review-platform" }).click();
  await expect(page.getByText(/\d+ resources · \d+ changing/)).toBeVisible();
  await expect(page.locator(".react-flow__node").filter({ hasText: "review-events" })).toHaveCount(0);
  await page.getByRole("button", { name: "Expand VPC review-platform" }).click();
  await expect(page.locator(".react-flow__node").filter({ hasText: "review-events" }).first()).toBeVisible();
});

test("the module view keeps the same resources", async ({ page }) => {
  await open(page, REPORTS.sample);
  await page.getByRole("radio", { name: "Modules" }).click();
  await expect(page.locator(".react-flow__node").filter({ hasText: "module.streaming" }).first()).toBeVisible();
  await page.getByRole("button", { name: /review-events/ }).first().click();
  await expect(page.getByRole("complementary", { name: /Inspector: review-events/ })).toBeVisible();
  await page.getByRole("radio", { name: "Architecture" }).click();
  await expect(page.getByRole("complementary", { name: /Inspector: review-events/ })).toBeVisible();
});

test("a safe-share report shows no original names anywhere in the page", async ({ page }) => {
  const watch = await open(page, REPORTS.safeShare);
  await expect(page.getByText("Safe-share", { exact: true })).toBeVisible();
  await page.getByLabel("Show").selectOption("all");
  const text = await page.locator("body").innerText();
  for (const original of ["review-orders", "review-events", "review-platform", "aws-review", "111122223333", "fixture.invalid"]) {
    expect(text).not.toContain(original);
  }
  await page.getByRole("button", { name: /^MSK cluster 1/ }).first().click();
  await expect(page.getByRole("complementary", { name: /Inspector: MSK cluster 1/ })).toBeVisible();
  expectCleanAndOffline(watch, REPORTS.safeShare);
});

test("icons from --icons render offline under the CSP, with symbols as the fallback", async ({ page }) => {
  const watch = await open(page, REPORTS.icons);
  const nat = page.locator(".react-flow__node").filter({ hasText: "NAT gateway main" }).first();
  const icon = nat.locator("img.resource-icon");
  await expect(icon).toHaveAttribute("src", /^data:image\/svg\+xml;base64,/);
  await expect(icon).toHaveAttribute("alt", "");
  expect(await icon.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true);
  // Security groups have no official icon and keep tfviz's symbol.
  const group = page.locator(".react-flow__node").filter({ hasText: "aws_security_group" }).first();
  await expect(group.locator("svg")).toHaveCount(1);
  await expect(group.locator("img")).toHaveCount(0);
  expectCleanAndOffline(watch, REPORTS.icons);
});

test("a state report lists every resource, since state has no changes", async ({ page }) => {
  const watch = await open(page, REPORTS.state);
  const listed = page.locator("button[data-resource]");
  await expect(listed.first()).toBeVisible();
  const total = Number((await page.getByText(/resources in this report/).innerText()).match(/\d+/)?.[0]);
  expect(await listed.count()).toBe(total);
  await expect(page.getByRole("option", { name: "Changes only" })).toHaveCount(0);
  await expect(page.getByText("No resources match these filters.")).toHaveCount(0);
  expectCleanAndOffline(watch, REPORTS.state);
});

test("relationship lines are drawn beneath resource cards", async ({ page }) => {
  await open(page, REPORTS.sample);
  const zOf = (selector: string) => page.locator(selector).first().evaluate((el) => Number(getComputedStyle(el).zIndex));
  const [edgeZ, cardZ, groupZ] = [await zOf(".react-flow__edges svg"), await zOf(".react-flow__node-resource"), await zOf(".react-flow__node-group")];
  expect(edgeZ).toBeGreaterThan(groupZ);
  expect(edgeZ).toBeLessThanOrEqual(cardZ);
  // Our group box draws the border; React Flow's default one must not add a second.
  expect(await page.locator(".react-flow__node-group").first().evaluate((el) => getComputedStyle(el).borderTopStyle)).toBe("none");
});
