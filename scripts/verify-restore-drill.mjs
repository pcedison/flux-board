import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { chromium, firefox, webkit } from "playwright";

const baseURL = normalizeBaseUrl(process.env.BASE_URL ?? "http://127.0.0.1:8080");
const password = process.env.FLUX_PASSWORD ?? process.env.APP_PASSWORD ?? "";
const browserName = normalizeBrowserName(
  process.env.PLAYWRIGHT_BROWSER ?? process.env.SMOKE_BROWSER ?? "chromium"
);
const headless = parseBoolean(process.env.HEADLESS, true);
const slowMo = parseInteger(process.env.SLOW_MO, 0);
const requestTimeoutMs = parseInteger(process.env.REQUEST_TIMEOUT_MS, 15000);
const drillTimeoutMs = parseInteger(process.env.RESTORE_DRILL_TIMEOUT_MS, 120000);
const viewport = { width: 1440, height: 960 };
const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const resultsDir =
  process.env.TEST_RESULTS_DIR ??
  path.join("test-results", "restore-drill", `verify-restore-drill-browser-${timestamp}`);

if (!password) {
  console.error("Missing FLUX_PASSWORD (or APP_PASSWORD) env var.");
  process.exit(1);
}

await mkdir(resultsDir, { recursive: true });

const overallTimeout = setTimeout(() => {
  console.error(`Restore drill browser verification timed out after ${drillTimeoutMs}ms.`);
  process.exit(1);
}, drillTimeoutMs);
if (typeof overallTimeout.unref === "function") {
  overallTimeout.unref();
}

const browserType = resolveBrowserType(browserName);
const browser = await browserType.launch({
  headless,
  slowMo: slowMo > 0 ? slowMo : undefined,
});

const context = await browser.newContext({ baseURL });
const page = await context.newPage();
page.setDefaultTimeout(requestTimeoutMs);
page.setDefaultNavigationTimeout(requestTimeoutMs);
await page.setViewportSize(viewport);

const summary = {
  baseURL,
  browser: browserName,
  resultsDir,
  checkedRoutes: ["/login", "/board", "/settings", "/api/export"],
  status: "failed",
};

try {
  logStep("Open login");
  await page.goto(`${baseURL}/login`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();
  await page.screenshot({ path: path.join(resultsDir, "01-login.png"), fullPage: true });

  logStep("Authenticate");
  const loginResponse = page.waitForResponse(
    (response) => response.url().endsWith("/api/auth/login") && response.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  const loginResult = await loginResponse;
  assert(loginResult.status() === 200, `Login failed with ${loginResult.status()}`);
  await page.waitForURL(/\/board$/);
  await page.getByRole("heading", { name: "New task" }).waitFor();

  const session = await requestJson(page, "/api/auth/me", requestTimeoutMs);
  assert(session.status === 200 && session.body?.authenticated === true, `Expected live auth session, got ${JSON.stringify(session)}`);
  await writeArtifact("auth-session.json", session.body);

  const boardSnapshot = await requestJson(page, "/api/tasks", requestTimeoutMs);
  assert(
    boardSnapshot.status === 200 && Array.isArray(boardSnapshot.body?.tasks),
    `Expected /api/tasks to return a task array, got ${JSON.stringify(boardSnapshot)}`
  );
  await writeArtifact("board-snapshot.json", boardSnapshot.body);
  await page.screenshot({ path: path.join(resultsDir, "02-board.png"), fullPage: true });

  logStep("Open settings");
  await page.goto(`${baseURL}/settings`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Backup & restore" }).waitFor();

  const settings = await requestJson(page, "/api/settings", requestTimeoutMs);
  assert(settings.status === 200 && isObject(settings.body), `Expected /api/settings to return an object, got ${JSON.stringify(settings)}`);
  await writeArtifact("settings.json", settings.body);
  await page.screenshot({ path: path.join(resultsDir, "03-settings.png"), fullPage: true });

  logStep("Capture export");
  const exportedBundle = await requestJson(page, "/api/export", requestTimeoutMs);
  assert(
    exportedBundle.status === 200 && isValidExportBundle(exportedBundle.body),
    `Expected /api/export to return a valid export bundle, got ${JSON.stringify(exportedBundle)}`
  );
  await writeArtifact("export-bundle.json", exportedBundle.body);

  summary.archiveRetentionDays = settings.body.archiveRetentionDays;
  summary.tasks = boardSnapshot.body.tasks.length;
  summary.archived = Array.isArray(exportedBundle.body.archived) ? exportedBundle.body.archived.length : 0;
  summary.exportVersion = exportedBundle.body.version;
  summary.exportedAt = exportedBundle.body.exportedAt;
  summary.status = "passed";
  console.log(`Restore drill browser verification completed successfully. Results: ${resultsDir}`);
} catch (error) {
  await page.screenshot({ path: path.join(resultsDir, "failure.png"), fullPage: true }).catch(() => {});
  summary.error = error instanceof Error ? error.message : String(error);
  console.error(summary.error);
  process.exitCode = 1;
} finally {
  await writeArtifact("summary.json", summary).catch(() => {});
  await browser.close();
  clearTimeout(overallTimeout);
}

async function requestJson(page, url, timeoutMs) {
  return page.evaluate(
    async ({ url, timeoutMs }) => {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(`request timed out after ${timeoutMs}ms`), timeoutMs);
      try {
        const response = await fetch(url, {
          credentials: "include",
          headers: { Accept: "application/json" },
          signal: controller.signal,
        });
        const body = await response.json().catch(() => null);
        return { status: response.status, ok: response.ok, body };
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        return { status: 0, ok: false, body: { error: message } };
      } finally {
        clearTimeout(timeout);
      }
    },
    { url, timeoutMs }
  );
}

async function writeArtifact(filename, value) {
  await writeFile(path.join(resultsDir, filename), JSON.stringify(value, null, 2), "utf8");
}

function isValidExportBundle(value) {
  return (
    isObject(value) &&
    typeof value.version === "string" &&
    value.version.length > 0 &&
    typeof value.exportedAt === "number" &&
    Number.isFinite(value.exportedAt) &&
    isObject(value.settings) &&
    Object.hasOwn(value.settings, "archiveRetentionDays") &&
    Array.isArray(value.tasks) &&
    Array.isArray(value.archived)
  );
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
        `Unsupported PLAYWRIGHT_BROWSER ${JSON.stringify(name)}. Expected one of chromium, firefox, webkit.`
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
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function isObject(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function logStep(step) {
  console.log(`\n=== ${step} ===`);
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}
