import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { chromium, firefox, webkit } from "playwright";

const baseURL = normalizeBaseUrl(process.env.BASE_URL ?? "http://127.0.0.1:8080");
const setupPassword = process.env.FLUX_SETUP_PASSWORD ?? process.env.FLUX_PASSWORD ?? "";
const browserName = normalizeBrowserName(
  process.env.PLAYWRIGHT_BROWSER ?? process.env.SMOKE_BROWSER ?? "chromium"
);
const headless = parseBoolean(process.env.HEADLESS, true);
const slowMo = parseInteger(process.env.SLOW_MO, 0);
const requestTimeoutMs = parseInteger(process.env.REQUEST_TIMEOUT_MS, 15000);
const smokeTimeoutMs = parseInteger(process.env.SMOKE_TIMEOUT_MS, 120000);
const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const resultsDir =
  process.env.TEST_RESULTS_DIR ??
  path.join("test-results", "e2e", `setup-smoke-${timestamp}`);

if (!setupPassword) {
  console.error("Missing FLUX_SETUP_PASSWORD (or FLUX_PASSWORD) env var.");
  process.exit(1);
}

await mkdir(resultsDir, { recursive: true });

const overallTimeout = setTimeout(() => {
  console.error(`Setup smoke timed out after ${smokeTimeoutMs}ms.`);
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
const context = await browser.newContext({ baseURL });
const page = await context.newPage();
page.setDefaultTimeout(requestTimeoutMs);
page.setDefaultNavigationTimeout(requestTimeoutMs);

try {
  logStep("Open first-run setup");
  await page.goto(baseURL, { waitUntil: "domcontentloaded" });
  await page.waitForURL(/\/setup$/);
  await page.getByRole("heading", { name: "Set the board password" }).waitFor();
  await page.screenshot({ path: path.join(resultsDir, "01-setup.png"), fullPage: true });

  logStep("Bootstrap");
  const bootstrapResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/bootstrap/setup") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByLabel("New password").fill(setupPassword);
  await page.getByLabel("Confirm password").fill(setupPassword);
  await page.getByRole("button", { name: "Finish setup" }).click();
  const bootstrapResult = await bootstrapResponse;
  assertStatus(bootstrapResult.status() === 200, `Bootstrap failed with ${bootstrapResult.status()}`);
  await page.waitForURL(/\/board$/);
  await page.getByRole("heading", { name: "New task" }).waitFor();

  const bootstrapStatus = await requestJson(page, "/api/bootstrap/status");
  assertStatus(
    bootstrapStatus.status === 200 && bootstrapStatus.body?.needsSetup === false,
    `Expected bootstrap status to be complete, got ${JSON.stringify(bootstrapStatus)}`
  );

  logStep("Create task");
  const setupTaskTitle = `Setup smoke ${Date.now()}`;
  await createTaskFromBoard(page, {
    due: "2026-05-01",
    note: "Verify setup-first bootstrap and follow-up sign-in.",
    title: setupTaskTitle,
  });
  await page.locator("article", { hasText: setupTaskTitle }).first().waitFor({ state: "visible", timeout: 10000 });

  logStep("Logout");
  await page.getByRole("button", { name: "Sign out" }).click();
  await page.waitForURL(/\/login$/);
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();

  logStep("Sign in with the new password");
  const loginResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/auth/login") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByLabel("Password").fill(setupPassword);
  await page.getByRole("button", { name: "Sign in" }).click();
  const loginResult = await loginResponse;
  assertStatus(loginResult.status() === 200, `Login after setup failed with ${loginResult.status()}`);
  await page.waitForURL(/\/board$/);
  await page.locator("article", { hasText: setupTaskTitle }).first().waitFor({ state: "visible", timeout: 10000 });
  await page.screenshot({ path: path.join(resultsDir, "02-board-after-setup.png"), fullPage: true });

  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        passwordSeedMode: "browser-bootstrap",
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
        passwordSeedMode: "browser-bootstrap",
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

async function requestJson(page, url, init = {}) {
  return page.evaluate(
    async ({ url, init, requestTimeoutMs }) => {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(`request timed out after ${requestTimeoutMs}ms`), requestTimeoutMs);
      try {
        const response = await fetch(url, {
          credentials: "include",
          signal: controller.signal,
          ...init,
        });
        const contentType = response.headers.get("content-type") || "";
        const body = contentType.includes("application/json")
          ? await response.json().catch(() => null)
          : await response.text().catch(() => null);
        return { status: response.status, ok: response.ok, body };
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        return { status: 0, ok: false, body: { error: message } };
      } finally {
        clearTimeout(timeout);
      }
    },
    { url, init, requestTimeoutMs }
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

async function createTaskFromBoard(page, { title, due, note }) {
  await page.getByLabel("Title").fill(title);
  await page.getByLabel("Due date").fill(due);
  await page.getByLabel("Note").fill(note);
  const createResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/tasks") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByRole("button", { name: "Create task" }).click();
  const response = await createResponse;
  assertStatus(response.status() === 201, `Create task failed with ${response.status()}`);
}
