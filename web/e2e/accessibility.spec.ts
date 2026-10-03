import { expect, test } from "@playwright/test";
import { REPORTS, fileUrl } from "./paths";

test.beforeEach(async ({ page }) => {
  await page.context().route(/^(?!file:)/, (route) => route.abort());
});

test("changes can be reviewed with the keyboard alone", async ({ page, browserName }) => {
  await page.goto(fileUrl(REPORTS.sample));
  await page.getByPlaceholder("Search resources").focus();
  await page.keyboard.type("review");
  if (browserName === "chromium") {
    // Firefox and Safari on macOS move Tab between text fields only unless
    // full keyboard access is enabled, so tab order is checked in Chromium.
    await page.keyboard.press("Tab"); // Show
    await page.keyboard.press("Tab"); // Domain
    await page.keyboard.press("Tab"); // first resource
  } else {
    await page.locator("button[data-resource]").first().focus();
  }
  await expect(page.locator("button[data-resource]:focus")).toHaveCount(1);
  await page.keyboard.press("ArrowDown");
  const second = page.locator("button[data-resource]:focus");
  const label = (await second.locator(".resource-list__label").innerText()).trim();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("complementary", { name: `Inspector: ${label}` })).toBeVisible();
});

test("the relationship trace stops when reduced motion is preferred", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto(fileUrl(REPORTS.sample));
  await page.getByRole("button", { name: /review-events/ }).first().click();
  const pulse = page.locator(".trace-edge__pulse").first();
  await expect(pulse).toBeAttached();
  expect(await pulse.evaluate((el) => getComputedStyle(el).animationDuration)).toBe("0s");
});

test("the reduce-motion control stops the trace too", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "no-preference" });
  await page.goto(fileUrl(REPORTS.sample));
  await page.getByRole("checkbox", { name: "Reduce motion" }).check();
  await page.getByRole("button", { name: /review-events/ }).first().click();
  const pulse = page.locator(".trace-edge__pulse").first();
  await expect(pulse).toBeAttached();
  expect(await pulse.evaluate((el) => getComputedStyle(el).animationDuration)).toBe("0s");
});

test("controls have accessible names and the page has landmarks", async ({ page }) => {
  await page.goto(fileUrl(REPORTS.sample));
  await expect(page.locator(".react-flow__node").first()).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Resources" })).toBeVisible();
  await expect(page.getByRole("main", { name: "Diagram" })).toBeVisible();
  await expect(page.getByRole("complementary", { name: "Inspector" })).toBeVisible();
  const unnamed = await page.evaluate(() =>
    [...document.querySelectorAll("button, select, input")]
      .filter((el) => {
        const named = el.getAttribute("aria-label") || (el as HTMLElement).innerText?.trim() || el.closest("label")?.innerText.trim() || el.getAttribute("placeholder") || el.getAttribute("title");
        return !named;
      })
      .map((el) => el.outerHTML.slice(0, 80)),
  );
  expect(unnamed).toEqual([]);
});
