import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { chromium, firefox, webkit } from "playwright";

const baseURL = normalizeBaseUrl(process.env.BASE_URL ?? "http://127.0.0.1:8080");
const nextAliasURL = `${baseURL}/next`;
const password = process.env.FLUX_PASSWORD ?? process.env.APP_PASSWORD ?? "";
const browserName = normalizeBrowserName(
  process.env.PLAYWRIGHT_BROWSER ?? process.env.SMOKE_BROWSER ?? "chromium"
);
const headless = parseBoolean(process.env.HEADLESS, true);
const slowMo = parseInteger(process.env.SLOW_MO, 0);
const requestTimeoutMs = parseInteger(process.env.REQUEST_TIMEOUT_MS, 15000);
const smokeTimeoutMs = parseInteger(process.env.SMOKE_TIMEOUT_MS, 120000);
const desktopViewport = { width: 1440, height: 960 };
const mobileViewport = { width: 390, height: 844 };
const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const resultsDir =
  process.env.TEST_RESULTS_DIR ??
  path.join("test-results", "e2e", `next-preview-smoke-${timestamp}`);

if (!password) {
  console.error("Missing FLUX_PASSWORD (or APP_PASSWORD) env var.");
  process.exit(1);
}

await mkdir(resultsDir, { recursive: true });

const overallTimeout = setTimeout(() => {
  console.error(`Next alias smoke timed out after ${smokeTimeoutMs}ms.`);
  process.exit(1);
}, smokeTimeoutMs);
if (typeof overallTimeout.unref === "function") {
  overallTimeout.unref();
}

const browserType = resolveBrowserType(browserName);
const browser = await browserType.launch({
  headless,
  slowMo: slowMo > 0 ? slowMo : undefined,
});

const context = await browser.newContext({
  baseURL,
});

const page = await context.newPage();
page.setDefaultTimeout(requestTimeoutMs);
page.setDefaultNavigationTimeout(requestTimeoutMs);
await page.setViewportSize(desktopViewport);

try {
  logStep("Next alias redirect");
  await page.goto(`${nextAliasURL}/login`, { waitUntil: "domcontentloaded" });
  await page.waitForURL(/\/login$/);
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();
  await assertHorizontalLayout(
    page.locator(".auth-layout > .panel").first(),
    page.locator(".auth-layout > .panel").nth(1),
    "Login panels on desktop via /next"
  );
  await page.screenshot({ path: path.join(resultsDir, "01-next-alias-login.png"), fullPage: true });

  logStep("Canonical login after redirect");
  const loginResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/auth/login") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  const loginResult = await loginResponse;
  if (loginResult.status() !== 200) {
    throw new Error(`Alias login failed with ${loginResult.status()}`);
  }
  await page.waitForURL(/\/board$/);
  await page.getByRole("heading", { name: "Create task" }).waitFor();

  const smokeTaskTitle = `Next alias smoke task ${Date.now()}`;

  logStep("Create task");
  await page.getByLabel("Title").fill(smokeTaskTitle);
  await page.getByLabel("Due date").fill("2026-04-30");
  await page.getByLabel("Note").fill("Verify /next compatibility redirect after root runtime takeover.");
  await page.getByRole("button", { name: "Create task" }).click();
  await page.getByText(`Created ${smokeTaskTitle} in the queued lane.`).waitFor();

  logStep("Archive task");
  await page.getByRole("button", { name: `Archive ${smokeTaskTitle}` }).click();
  await page.getByText(`Archived ${smokeTaskTitle}.`).waitFor();
  await page.getByRole("button", { name: `Restore ${smokeTaskTitle}` }).waitFor();

  logStep("Restore task");
  await page.getByRole("button", { name: `Restore ${smokeTaskTitle}` }).click();
  await page.getByText(`Restored ${smokeTaskTitle} to queued.`).waitFor();

  logStep("Logout");
  await page.getByRole("button", { name: "Sign out" }).click();
  await page.waitForURL(/\/login$/);
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();

  logStep("Mobile alias redirect");
  await page.setViewportSize(mobileViewport);
  await page.goto(`${nextAliasURL}/login`, { waitUntil: "domcontentloaded" });
  await page.waitForURL(/\/login$/);
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();
  await assertVerticalLayout(
    page.locator(".auth-layout > .panel").first(),
    page.locator(".auth-layout > .panel").nth(1),
    "Login panels on mobile via /next"
  );

  logStep("Legacy rollback route");
  await page.goto(`${baseURL}/legacy/`, { waitUntil: "domcontentloaded" });
  await page.locator("#loginOverlay").waitFor({ state: "visible", timeout: 10000 });
  await page.screenshot({ path: path.join(resultsDir, "02-legacy-rollback.png"), fullPage: true });

  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        nextAliasURL,
        legacyURL: `${baseURL}/legacy/`,
        viewports: ["desktop", "mobile"],
        resultsDir,
        status: "passed",
      },
      null,
      2
    ),
    "utf8"
  );
} catch (error) {
  await page.screenshot({ path: path.join(resultsDir, "failure.png"), fullPage: true }).catch(() => {});
  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        nextAliasURL,
        legacyURL: `${baseURL}/legacy/`,
        viewports: ["desktop", "mobile"],
        resultsDir,
        status: "failed",
        error: error instanceof Error ? error.message : String(error),
      },
      null,
      2
    ),
    "utf8"
  ).catch(() => {});
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
} finally {
  await browser.close();
  clearTimeout(overallTimeout);
}

function normalizeBaseUrl(value) {
  const url = new URL(value);
  return url.toString().replace(/\/$/, "");
}

function normalizeBrowserName(value) {
  return String(value).trim().toLowerCase();
}

function resolveBrowserType(name) {
  switch (name) {
    case "chromium":
      return chromium;
    case "firefox":
      return firefox;
    case "webkit":
      return webkit;
    default:
      throw new Error(
        `Unsupported PLAYWRIGHT_BROWSER ${JSON.stringify(
          name
        )}. Expected one of chromium, firefox, webkit.`
      );
  }
}

function parseBoolean(value, fallback) {
  if (value == null) {
    return fallback;
  }
  const normalized = String(value).trim().toLowerCase();
  if (["1", "true", "yes", "on"].includes(normalized)) {
    return true;
  }
  if (["0", "false", "no", "off"].includes(normalized)) {
    return false;
  }
  return fallback;
}

function parseInteger(value, fallback) {
  if (value == null || value === "") {
    return fallback;
  }
  const parsed = Number.parseInt(String(value), 10);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function logStep(step) {
  console.log(`\n=== ${step} ===`);
}

function assertStatus(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function assertHorizontalLayout(firstLocator, secondLocator, label) {
  const [firstBox, secondBox] = await Promise.all([firstLocator.boundingBox(), secondLocator.boundingBox()]);
  assertStatus(firstBox != null, `${label}: first element is not visible.`);
  assertStatus(secondBox != null, `${label}: second element is not visible.`);
  assertStatus(secondBox.x > firstBox.x + 16, `${label} should flow side-by-side on wide viewports.`);
  assertStatus(
    Math.abs(secondBox.y - firstBox.y) < 32,
    `${label} should stay on the same row on wide viewports.`
  );
}

async function assertVerticalLayout(firstLocator, secondLocator, label) {
  const [firstBox, secondBox] = await Promise.all([firstLocator.boundingBox(), secondLocator.boundingBox()]);
  assertStatus(firstBox != null, `${label}: first element is not visible.`);
  assertStatus(secondBox != null, `${label}: second element is not visible.`);
  assertStatus(secondBox.y > firstBox.y + 16, `${label} should stack on narrow viewports.`);
  assertStatus(
    Math.abs(secondBox.x - firstBox.x) < 64,
    `${label} should stay aligned when stacked on narrow viewports.`
  );
}
