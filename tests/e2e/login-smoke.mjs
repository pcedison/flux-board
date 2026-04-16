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
const smokeTimeoutMs = parseInteger(process.env.SMOKE_TIMEOUT_MS, 120000);
const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const resultsDir =
  process.env.TEST_RESULTS_DIR ??
  path.join("test-results", "e2e", `login-smoke-${timestamp}`);

if (!password) {
  console.error("Missing FLUX_PASSWORD (or APP_PASSWORD) env var.");
  process.exit(1);
}

await mkdir(resultsDir, { recursive: true });

const overallTimeout = setTimeout(() => {
  console.error(`Smoke timed out after ${smokeTimeoutMs}ms.`);
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
page.on("console", (msg) => {
  const text = msg.text();
  if (msg.type() === "error" || msg.type() === "warning") {
    console.log(`[console:${msg.type()}] ${text}`);
  }
});
page.on("pageerror", (err) => {
  console.log(`[pageerror] ${err.message}`);
});
page.on("requestfailed", (req) => {
  console.log(`[requestfailed] ${req.method()} ${req.url()} ${req.failure()?.errorText ?? ""}`);
});
page.on("response", async (res) => {
  const url = res.url();
  if (
    url.endsWith("/api/auth/login") ||
    url.endsWith("/api/auth/me") ||
    url.endsWith("/api/auth/logout") ||
    url.includes("/api/tasks") ||
    url.includes("/api/archived")
  ) {
    console.log(`[response] ${res.request().method()} ${url} -> ${res.status()}`);
  }
});

const artifacts = [];

try {
  logStep("Navigate");
  await page.goto(baseURL, { waitUntil: "domcontentloaded" });
  await page.locator("#loginOverlay").waitFor({ state: "visible", timeout: 10000 });
  await page.screenshot({ path: path.join(resultsDir, "01-loaded.png"), fullPage: true });
  artifacts.push("01-loaded.png");

  logStep("Login");
  const overlay = page.locator("#loginOverlay");
  if (await overlay.isVisible()) {
    await page.locator("#passwordInput").fill(password);
    const loginResponse = page.waitForResponse(
      (res) => res.url().endsWith("/api/auth/login") && res.request().method() === "POST",
      { timeout: 10000 }
    );
    await page.locator("#loginOverlay button[type='submit']").click();
    const response = await loginResponse;
    if (response.status() !== 200) {
      const body = await safeReadJson(response);
      throw new Error(`Login failed with ${response.status()}: ${JSON.stringify(body)}`);
    }
    await page.waitForFunction(
      () => document.getElementById("loginOverlay")?.hidden === true,
      null,
      { timeout: 10000 }
    );
  }
  await page.screenshot({ path: path.join(resultsDir, "02-after-login.png"), fullPage: true });
  artifacts.push("02-after-login.png");

  logStep("Session");
  const session = await requestJson(page, "/api/auth/me");
  assertStatus(session.status === 200, `Expected /api/auth/me to return 200, got ${session.status}`);
  assertObject(session.body, "me response");
  assertStatus(session.body.authenticated === true, "Expected /api/auth/me to report authenticated=true.");
  assertStatus(
    (typeof session.body.expiresAt === "number" && Number.isFinite(session.body.expiresAt) && session.body.expiresAt > 0) ||
      (typeof session.body.expiresAt === "string" && session.body.expiresAt.length > 0),
    "Expected /api/auth/me to include expiresAt as a non-empty value."
  );

  logStep("Tasks");
  const tasks = await requestJson(page, "/api/tasks");
  assertStatus(tasks.status === 200, `Expected /api/tasks to return 200, got ${tasks.status}`);
  assertArray(tasks.body.tasks, "tasks");

  logStep("Archived");
  const archived = await requestJson(page, "/api/archived");
  assertStatus(archived.status === 200, `Expected /api/archived to return 200, got ${archived.status}`);
  assertArray(archived.body.tasks, "archived tasks");

  const smokeTaskTitle = `Smoke task ${Date.now()}`;

  logStep("Create task");
  const taskCreateResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/tasks") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.locator("#openModalBtn").click();
  await page.locator("#taskTitle").fill(smokeTaskTitle);
  await page.locator("#taskDue").fill("2026-04-30");
  await page.locator("#taskPriority").selectOption("high");
  await page.locator("#submitBtn").click();
  const createResponse = await taskCreateResponse;
  if (createResponse.status() !== 201) {
    const body = await safeReadJson(createResponse);
    throw new Error(`Task create failed with ${createResponse.status()}: ${JSON.stringify(body)}`);
  }
  const createdCard = page.locator(".task-card", { hasText: smokeTaskTitle }).first();
  await createdCard.waitFor({ state: "visible", timeout: 10000 });

  logStep("Archive task");
  const taskArchiveResponse = page.waitForResponse(
    (res) => res.url().includes("/api/tasks/") && res.request().method() === "DELETE",
    { timeout: 10000 }
  );
  await createdCard.locator(".card-menu-trigger").click();
  await createdCard.locator("button[data-action='archive']").click();
  const archiveResponse = await taskArchiveResponse;
  if (archiveResponse.status() !== 200) {
    const body = await safeReadJson(archiveResponse);
    throw new Error(`Task archive failed with ${archiveResponse.status()}: ${JSON.stringify(body)}`);
  }
  await createdCard.waitFor({ state: "detached", timeout: 10000 });

  logStep("Restore task");
  await page.locator("#archiveToggleBtn").click();
  const archivedItem = page.locator(".archive-item", { hasText: smokeTaskTitle }).first();
  await archivedItem.waitFor({ state: "visible", timeout: 10000 });
  const taskRestoreResponse = page.waitForResponse(
    (res) => res.url().includes("/api/archived/") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await archivedItem.locator("button[data-action='restore']").click();
  const restoreResponse = await taskRestoreResponse;
  if (restoreResponse.status() !== 200) {
    const body = await safeReadJson(restoreResponse);
    throw new Error(`Task restore failed with ${restoreResponse.status()}: ${JSON.stringify(body)}`);
  }
  await page.locator(".task-card", { hasText: smokeTaskTitle }).first().waitFor({ state: "visible", timeout: 10000 });

  logStep("Logout");
  const logout = await requestJson(page, "/api/auth/logout", { method: "POST" });
  assertStatus(logout.status === 200, `Expected /api/auth/logout to return 200, got ${logout.status}`);

  const postLogout = await requestJson(page, "/api/auth/me");
  assertStatus(postLogout.status === 401, `Expected /api/auth/me to return 401 after logout, got ${postLogout.status}`);

  await page.screenshot({ path: path.join(resultsDir, "03-after-logout.png"), fullPage: true });
  artifacts.push("03-after-logout.png");

  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        headless,
        slowMo,
        resultsDir,
        artifacts,
        status: "passed",
      },
      null,
      2
    ),
    "utf8"
  );

  console.log(`Smoke passed. Artifacts written to ${resultsDir}`);
} catch (error) {
  await page.screenshot({ path: path.join(resultsDir, "failure.png"), fullPage: true }).catch(() => {});
  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        headless,
        slowMo,
        resultsDir,
        artifacts,
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
        let body = null;
        const contentType = response.headers.get("content-type") || "";
        if (contentType.includes("application/json")) {
          body = await response.json().catch(() => null);
        } else {
          body = await response.text().catch(() => null);
        }
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

async function safeReadJson(response) {
  try {
    return await response.json();
  } catch {
    try {
      return await response.text();
    } catch {
      return null;
    }
  }
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

function assertArray(value, label) {
  if (!Array.isArray(value)) {
    throw new Error(`Expected ${label} to be an array.`);
  }
}

function assertObject(value, label) {
  if (value == null || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`Expected ${label} to be an object.`);
  }
}
