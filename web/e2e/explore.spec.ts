import { expect, test } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { fileURLToPath } from "node:url";
import { REPORTS } from "./paths";

const statePath = fileURLToPath(new URL("../../testdata/producer/terraform-1.16/state.json", import.meta.url));

/** Starts the real `tfviz explore` and resolves with the link it prints. */
function startExplorer(): Promise<{ child: ChildProcess; link: string }> {
  return new Promise((resolve, reject) => {
    const child = spawn(`${REPORTS.dir}tfviz`, ["explore", "--state", statePath], { stdio: ["ignore", "ignore", "pipe"] });
    let output = "";
    const timer = setTimeout(() => reject(new Error(`explorer did not start: ${output}`)), 15_000);
    child.stderr?.on("data", (chunk: Buffer) => {
      output += chunk.toString();
      const match = output.match(/http:\/\/127\.0\.0\.1:\d+\/\?token=[\w-]+/);
      if (match) {
        clearTimeout(timer);
        resolve({ child, link: match[0] });
      }
    });
    child.on("exit", (code) => reject(new Error(`explorer exited with ${code}: ${output}`)));
  });
}

let explorer: { child: ChildProcess; link: string };
let attacker: Server;

test.beforeAll(async () => {
  explorer = await startExplorer();
  attacker = createServer((_, res) => {
    res.setHeader("Content-Type", "text/html");
    res.end("<!doctype html><title>other site</title>");
  });
  await new Promise<void>((resolve) => attacker.listen(0, "127.0.0.1", resolve));
});

test.afterAll(() => {
  explorer.child.kill("SIGINT");
  attacker.close();
});

test("the explorer serves the interface and refreshes on request", async ({ page }) => {
  const problems: string[] = [];
  page.on("console", (msg) => msg.type() === "error" && problems.push(msg.text()));
  page.on("pageerror", (err) => problems.push(err.message));
  await page.goto(explorer.link);
  // The access token is exchanged for a cookie and removed from the URL.
  expect(page.url()).not.toContain("token=");
  await expect(page.getByText("Local explorer")).toBeVisible();
  await expect(page.locator(".react-flow__node").first()).toBeVisible();

  const reloaded = page.waitForEvent("load");
  await page.getByRole("button", { name: "Refresh" }).click();
  await reloaded;
  await expect(page.getByText("Local explorer")).toBeVisible();
  expect(problems).toEqual([]);
});

test("another local site cannot read the explorer", async ({ page }) => {
  // Sign in first so the browser holds the explorer's session cookie.
  await page.goto(explorer.link);
  const origin = new URL(explorer.link).origin;
  const { port } = attacker.address() as AddressInfo;
  await page.goto(`http://127.0.0.1:${port}/`);
  const outcome = await page.evaluate(async (target) => {
    const results: string[] = [];
    for (const [path, init] of [
      ["/", { credentials: "include" }],
      ["/refresh", { method: "POST", credentials: "include", headers: { "X-Tfviz-Token": "guess" } }],
    ] as const) {
      try {
        const res = await fetch(target + path, init as RequestInit);
        results.push(`${path}: read ${res.status} ${(await res.text()).includes("tfviz-report")}`);
      } catch {
        results.push(`${path}: blocked`);
      }
    }
    return results;
  }, origin);
  expect(outcome).toEqual(["/: blocked", "/refresh: blocked"]);
});
