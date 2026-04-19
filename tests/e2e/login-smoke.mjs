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
const desktopViewport = { width: 1440, height: 960 };
const mobileViewport = { width: 390, height: 844 };
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
await page.setViewportSize(desktopViewport);

try {
  logStep("Navigate");
  await page.goto(`${baseURL}/login`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();
  await assertHorizontalLayout(
    page.locator(".auth-layout > .panel").first(),
    page.locator(".auth-layout > .panel").nth(1),
    "Login panels on desktop"
  );
  await page.screenshot({ path: path.join(resultsDir, "01-login.png"), fullPage: true });

  logStep("Login");
  const loginResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/auth/login") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  const loginResult = await loginResponse;
  if (loginResult.status() !== 200) {
    throw new Error(`Login failed with ${loginResult.status()}`);
  }
  await page.waitForURL(/\/board$/);
  await page.getByRole("heading", { name: "New task" }).waitFor();
  await assertStatus(
    (await page.locator(".board-grid > .lane").count()) === 3,
    "Expected three board lanes on the React runtime."
  );
  await assertHorizontalLayout(
    page.locator(".board-grid > .lane").first(),
    page.locator(".board-grid > .board-side-panel"),
    "Board layout on desktop"
  );

  logStep("Session");
  const session = await requestJson(page, "/api/auth/me");
  assertStatus(session.status === 200, `Expected /api/auth/me to return 200, got ${session.status}`);

  const smokeTaskTitle = `Smoke task ${Date.now()}`;

  logStep("Mobile layout");
  await page.setViewportSize(mobileViewport);
  await assertVerticalLayout(
    page.locator(".board-grid > .lane").first(),
    page.locator(".board-grid > .board-side-panel"),
    "Board layout on mobile"
  );
  await assertVerticalLayout(
    page.locator(".board-side-panel .field-grid > div").first(),
    page.locator(".board-side-panel .field-grid > div").nth(1),
    "Composer field grid on mobile"
  );

  logStep("Create task");
  const createdCard = await createTaskFromBoard(page, {
    due: "2026-04-30",
    note: "Verify root runtime ownership after W7 2-B.",
    title: smokeTaskTitle,
  });

  logStep("Archive task");
  await archiveTaskFromBoard(page, createdCard, smokeTaskTitle);
  await page.locator(".panel-tabs").getByRole("button", { name: "Archive" }).click();
  const restoreButton = page.getByRole("button", { name: `Restore ${smokeTaskTitle}` });
  await restoreButton.waitFor();

  logStep("Restore task");
  const restoreResponse = page.waitForResponse(
    (res) => res.url().includes("/api/archived/") && res.url().endsWith("/restore") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await restoreButton.click();
  const restoreResult = await restoreResponse;
  assertStatus(restoreResult.ok(), `Restore task failed with ${restoreResult.status()}`);
  await createdCard.waitFor({ state: "visible", timeout: 10000 });

  logStep("Logout");
  await page.getByRole("button", { name: "Sign out" }).click();
  await page.waitForURL(/\/login$/);
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();
  await assertPanelsVisible(
    page.locator(".auth-layout > .panel").first(),
    page.locator(".auth-layout > .panel").nth(1),
    "Login panels on mobile"
  );

  const postLogout = await requestJson(page, "/api/auth/me");
  assertStatus(postLogout.status === 401, `Expected /api/auth/me to return 401 after logout, got ${postLogout.status}`);

  await page.screenshot({ path: path.join(resultsDir, "02-logout.png"), fullPage: true });

  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        canonicalRuntime: "/login -> /board",
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
        canonicalRuntime: "/login -> /board",
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
  const createdCard = page.locator("article", { hasText: title }).first();
  await createdCard.waitFor({ state: "visible", timeout: 10000 });
  return createdCard;
}

async function archiveTaskFromBoard(page, card, title) {
  await card.click();
  await page.getByRole("heading", { name: "Task details" }).waitFor();
  const archiveResponse = page.waitForResponse(
    (res) => res.url().includes("/api/tasks/") && res.request().method() === "DELETE",
    { timeout: 10000 }
  );
  await page.getByRole("button", { name: "Archive task" }).click();
  const response = await archiveResponse;
  assertStatus(response.ok(), `Archive task failed with ${response.status()}`);
  await page.getByText(`Archived ${title}.`).waitFor();
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

async function assertPanelsVisible(firstLocator, secondLocator, label) {
  const [firstBox, secondBox] = await Promise.all([firstLocator.boundingBox(), secondLocator.boundingBox()]);
  assertStatus(firstBox != null, `${label}: first element is not visible.`);
  assertStatus(secondBox != null, `${label}: second element is not visible.`);
}
