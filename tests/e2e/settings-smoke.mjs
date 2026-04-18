import { randomUUID } from "node:crypto";
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
const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const resultsDir =
  process.env.TEST_RESULTS_DIR ??
  path.join("test-results", "e2e", `settings-smoke-${timestamp}`);
const originalBundlePath = path.join(resultsDir, "original-export.json");
const exportedBundlePath = path.join(resultsDir, "exported-settings-bundle.json");
const invalidBundlePath = path.join(resultsDir, "invalid-import-bundle.json");
const mutatedBundlePath = path.join(resultsDir, "mutated-import-bundle.json");
const rotatedPassword = `settings-smoke-${Date.now()}-pw`;
const importedTaskTitle = `Settings import smoke ${Date.now()}`;
const importedTaskID = `settings-smoke-${randomUUID()}`;

if (!password) {
  console.error("Missing FLUX_PASSWORD (or APP_PASSWORD) env var.");
  process.exit(1);
}

await mkdir(resultsDir, { recursive: true });

const overallTimeout = setTimeout(() => {
  console.error(`Settings smoke timed out after ${smokeTimeoutMs}ms.`);
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
  acceptDownloads: true,
  baseURL,
});
const page = await context.newPage();
page.setDefaultTimeout(requestTimeoutMs);
page.setDefaultNavigationTimeout(requestTimeoutMs);
await page.setViewportSize(desktopViewport);

let secondaryContext = null;
let secondaryPage = null;
let originalBundle = null;
let activePassword = password;
let originalStateRestored = true;
let originalPasswordRestored = true;
let rollback = { attempted: false, steps: [] };
let initialSessions = [];
let invalidImportRejected = false;

try {
  logStep("Login primary session");
  await openLogin(page, baseURL);
  await login(page, password);

  logStep("Open settings");
  await openSettings(page, baseURL);
  await page.screenshot({ path: path.join(resultsDir, "01-settings.png"), fullPage: true });

  const initialSessionsResponse = await requestJson(page, "/api/settings/sessions");
  assertStatus(
    initialSessionsResponse.status === 200,
    `Expected /api/settings/sessions to return 200, got ${initialSessionsResponse.status}`
  );
  initialSessions = await ensureSingleCurrentSession(page, baseURL, initialSessionsResponse.body?.sessions ?? []);

  logStep("Capture original board export");
  const originalExportResponse = await requestJson(page, "/api/export");
  assertStatus(
    originalExportResponse.status === 200 && originalExportResponse.body,
    `Expected /api/export to return 200, got ${originalExportResponse.status}`
  );
  originalBundle = originalExportResponse.body;
  await writeFile(originalBundlePath, JSON.stringify(originalBundle, null, 2), "utf8");

  logStep("Reject malformed import without mutating state");
  const invalidBundle = buildInvalidBundle(originalBundle);
  await writeFile(invalidBundlePath, JSON.stringify(invalidBundle, null, 2), "utf8");
  await importBundleExpectFailure(page, invalidBundlePath, {
    expectedMessage: "export version is required",
    expectedStatus: 400,
  });
  invalidImportRejected = true;
  const stateAfterInvalidImport = await requestJson(page, "/api/export");
  assertStatus(
    stateAfterInvalidImport.status === 200 &&
      bundlesMatchForState(originalBundle, stateAfterInvalidImport.body),
    "Expected malformed import rejection to preserve the current board snapshot."
  );
  await page.screenshot({
    path: path.join(resultsDir, "02-invalid-import-rejected.png"),
    fullPage: true,
  });

  logStep("Create secondary session");
  secondaryContext = await browser.newContext({ baseURL });
  secondaryPage = await secondaryContext.newPage();
  secondaryPage.setDefaultTimeout(requestTimeoutMs);
  secondaryPage.setDefaultNavigationTimeout(requestTimeoutMs);
  await secondaryPage.setViewportSize(desktopViewport);
  await openLogin(secondaryPage, baseURL);
  await login(secondaryPage, password);

  await openSettings(page, baseURL);
  const sessionsAfterSecondLogin = await requestJson(page, "/api/settings/sessions");
  const currentSessions = sessionsAfterSecondLogin.body?.sessions ?? [];
  assertStatus(
    sessionsAfterSecondLogin.status === 200 && currentSessions.length === initialSessions.length + 1,
    `Expected one extra session after secondary login, got ${JSON.stringify(sessionsAfterSecondLogin)}`
  );
  const secondarySession = findExtraSession(initialSessions, currentSessions);
  const expectedSessionSummary = `${currentSessions.length} active ${
    currentSessions.length === 1 ? "session" : "sessions"
  }`;
  await page.getByText(expectedSessionSummary).waitFor();

  logStep("Update archive policy");
  const originalRetentionDays = originalBundle.settings?.archiveRetentionDays ?? null;
  const updatedRetentionDays = selectRetentionDays([originalRetentionDays]);
  await page.getByLabel("Auto-delete archived cards").check();
  await page.getByLabel("Retention (days)").fill(String(updatedRetentionDays));
  await page.getByRole("button", { name: "Save archive policy" }).click();
  await page
    .getByText(`Archived cards will auto-delete after ${updatedRetentionDays} days.`)
    .waitFor();
  const updatedSettings = await requestJson(page, "/api/settings");
  assertStatus(
    updatedSettings.status === 200 &&
      updatedSettings.body?.archiveRetentionDays === updatedRetentionDays,
    `Expected archive retention ${updatedRetentionDays}, got ${JSON.stringify(updatedSettings)}`
  );

  logStep("Export board data");
  const exportResponsePromise = page.waitForResponse(
    (res) => res.url().endsWith("/api/export") && res.request().method() === "GET",
    { timeout: 10000 }
  );
  await page.getByRole("button", { name: "Export board data" }).click();
  const exportResponse = await exportResponsePromise;
  assertStatus(exportResponse.status() === 200, `Export failed with ${exportResponse.status()}`);
  await page.getByText("Export completed.").waitFor();
  const exportedBundle = await exportResponse.json();
  assertStatus(
    exportedBundle.settings?.archiveRetentionDays === updatedRetentionDays,
    `Expected exported bundle retention ${updatedRetentionDays}, got ${JSON.stringify(
      exportedBundle.settings
    )}`
  );
  await writeFile(exportedBundlePath, JSON.stringify(exportedBundle, null, 2), "utf8");

  logStep("Import modified bundle");
  const importRetentionDays = selectRetentionDays([originalRetentionDays, updatedRetentionDays]);
  const mutatedBundle = buildMutatedBundle(exportedBundle, importRetentionDays);
  await writeFile(mutatedBundlePath, JSON.stringify(mutatedBundle, null, 2), "utf8");
  originalStateRestored = false;
  await importBundle(page, mutatedBundlePath);
  const importedSettings = await requestJson(page, "/api/settings");
  assertStatus(
    importedSettings.status === 200 &&
      importedSettings.body?.archiveRetentionDays === importRetentionDays,
    `Expected imported archive retention ${importRetentionDays}, got ${JSON.stringify(
      importedSettings
    )}`
  );
  await page.goto(`${baseURL}/board`, { waitUntil: "domcontentloaded" });
  await page.locator("article", { hasText: importedTaskTitle }).first().waitFor({
    state: "visible",
    timeout: 10000,
  });
  await page.screenshot({ path: path.join(resultsDir, "03-imported-board.png"), fullPage: true });
  await openSettings(page, baseURL);

  logStep("Revoke secondary session");
  const secondarySessionRow = await locateSavedSessionRow(page, secondarySession);
  await secondarySessionRow.getByRole("button", { name: "Revoke" }).click();
  const sessionCountAfterRevoke = initialSessions.length;
  const revokedSessionSummary = `${sessionCountAfterRevoke} active ${
    sessionCountAfterRevoke === 1 ? "session" : "sessions"
  }`;
  await page.getByText(revokedSessionSummary).waitFor();
  const sessionsAfterRevoke = await requestJson(page, "/api/settings/sessions");
  assertStatus(
    sessionsAfterRevoke.status === 200 &&
      (sessionsAfterRevoke.body?.sessions ?? []).length === sessionCountAfterRevoke,
    `Expected ${sessionCountAfterRevoke} active session after revoke, got ${JSON.stringify(
      sessionsAfterRevoke
    )}`
  );
  const revokedAuth = await requestJson(secondaryPage, "/api/auth/me");
  assertStatus(
    revokedAuth.status === 401,
    `Expected revoked secondary session to be unauthorized, got ${JSON.stringify(revokedAuth)}`
  );
  await page.screenshot({ path: path.join(resultsDir, "04-session-revoked.png"), fullPage: true });

  logStep("Restore original board export");
  await importBundle(page, originalBundlePath);
  originalStateRestored = true;
  const restoredSettings = await requestJson(page, "/api/settings");
  assertStatus(
    restoredSettings.status === 200 &&
      restoredSettings.body?.archiveRetentionDays === originalRetentionDays,
    `Expected restored archive retention ${JSON.stringify(
      originalRetentionDays
    )}, got ${JSON.stringify(restoredSettings)}`
  );
  const restoredTasks = await requestJson(page, "/api/tasks");
  assertStatus(
    restoredTasks.status === 200 &&
      !(restoredTasks.body?.tasks ?? []).some((task) => task.title === importedTaskTitle),
    "Expected imported smoke task to be removed after restoring the original export."
  );

  logStep("Rotate password");
  originalPasswordRestored = false;
  await changePasswordViaSettings(page, password, rotatedPassword);
  activePassword = rotatedPassword;
  await page.getByText("Password updated. Other sessions were signed out.").waitFor();
  await page.screenshot({ path: path.join(resultsDir, "05-password-rotated.png"), fullPage: true });

  logStep("Re-login with rotated password");
  await signOut(page);
  const failedLogin = await submitLogin(page, password);
  assertStatus(
    failedLogin.status() === 401,
    `Expected old password login to fail with 401, got ${failedLogin.status()}`
  );
  await page.getByText("invalid password").waitFor();
  await login(page, rotatedPassword);

  logStep("Restore original password");
  await openSettings(page, baseURL);
  await changePasswordViaSettings(page, rotatedPassword, password);
  activePassword = password;
  originalPasswordRestored = true;
  await page.getByText("Password updated. Other sessions were signed out.").waitFor();

  logStep("Re-login with original password");
  await signOut(page);
  await login(page, password);
  await openSettings(page, baseURL);
  const finalSessions = await requestJson(page, "/api/settings/sessions");
  assertStatus(
    finalSessions.status === 200 &&
      (finalSessions.body?.sessions ?? []).length === 1 &&
      finalSessions.body?.sessions?.[0]?.current === true,
    `Expected a single current session at the end of smoke, got ${JSON.stringify(finalSessions)}`
  );
  await page.screenshot({ path: path.join(resultsDir, "06-settings-restored.png"), fullPage: true });

  logStep("Logout");
  await signOut(page);
  const postLogout = await requestJson(page, "/api/auth/me");
  assertStatus(
    postLogout.status === 401,
    `Expected /api/auth/me to return 401 after logout, got ${postLogout.status}`
  );

  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        canonicalRuntime: "/login -> /settings",
        importedTaskTitle,
        invalidImportRejected,
        restoration: {
          boardExport: originalStateRestored ? "restored" : "dirty",
          password: originalPasswordRestored ? "restored" : "dirty",
        },
        resultsDir,
        rollback,
        status: "passed",
      },
      null,
      2
    ),
    "utf8"
  );
} catch (error) {
  rollback = await attemptRollback({
    activePassword,
    baseURL,
    originalBundlePath,
    originalPassword: password,
    originalPasswordRestored,
    originalStateRestored,
    page,
  });
  if (rollback.steps.includes("Restored original board export.")) {
    originalStateRestored = true;
  }
  if (rollback.steps.includes("Restored original password.")) {
    originalPasswordRestored = true;
  }
  await page
    .screenshot({ path: path.join(resultsDir, "failure.png"), fullPage: true })
    .catch(() => {});
  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        canonicalRuntime: "/login -> /settings",
        importedTaskTitle,
        invalidImportRejected,
        restoration: {
          boardExport: originalStateRestored ? "restored" : "dirty",
          password: originalPasswordRestored ? "restored" : "dirty",
        },
        resultsDir,
        rollback,
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
  await secondaryContext?.close().catch(() => {});
  await browser.close();
  clearTimeout(overallTimeout);
}

async function attemptRollback({
  activePassword,
  baseURL,
  originalBundlePath,
  originalPassword,
  originalPasswordRestored,
  originalStateRestored,
  page,
}) {
  const steps = [];

  if (!page || page.isClosed()) {
    return { attempted: false, steps };
  }

  try {
    await ensureLoggedIn(page, baseURL, activePassword);
    await openSettings(page, baseURL);

    if (!originalStateRestored) {
      await importBundle(page, originalBundlePath);
      steps.push("Restored original board export.");
    }

    if (!originalPasswordRestored && activePassword !== originalPassword) {
      await changePasswordViaSettings(page, activePassword, originalPassword);
      steps.push("Restored original password.");
    }
  } catch (error) {
    steps.push(`Rollback failed: ${error instanceof Error ? error.message : String(error)}`);
  }

  return {
    attempted: steps.length > 0,
    steps,
  };
}

async function ensureSingleCurrentSession(page, baseURL, sessions) {
  void page;
  void baseURL;
  const currentSessions = sessions.filter((session) => session.current);
  const nonCurrentSessions = sessions.filter((session) => !session.current);
  assertStatus(
    currentSessions.length === 1 && nonCurrentSessions.length === 0,
    `Settings smoke requires an isolated runtime with exactly one current session before creating a second browser session. Got ${JSON.stringify(
      sessions
    )}`
  );
  return sessions;
}

async function ensureLoggedIn(page, baseURL, candidatePassword) {
  const session = await requestJson(page, "/api/auth/me");
  if (session.status === 200) {
    return;
  }
  await openLogin(page, baseURL);
  await login(page, candidatePassword);
}

async function openLogin(page, baseURL) {
  await page.goto(`${baseURL}/login`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();
}

async function openSettings(page, baseURL) {
  await page.goto(`${baseURL}/settings`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Archive policy" }).waitFor();
  await page.getByRole("heading", { name: "Sessions" }).waitFor();
  await page.getByRole("heading", { name: "Backup & import" }).waitFor();
}

async function submitLogin(page, nextPassword) {
  const loginResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/auth/login") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByLabel("Password").fill(nextPassword);
  await page.getByRole("button", { name: "Sign in" }).click();
  return loginResponse;
}

async function login(page, nextPassword) {
  const loginResult = await submitLogin(page, nextPassword);
  assertStatus(loginResult.status() === 200, `Login failed with ${loginResult.status()}`);
  await page.waitForURL((url) => !url.pathname.endsWith("/login"), { timeout: 10000 });
  const currentPath = new URL(page.url()).pathname;
  if (currentPath.endsWith("/board")) {
    await page.getByRole("heading", { name: "Create task" }).waitFor();
    return;
  }
  if (currentPath.endsWith("/settings")) {
    await page.getByRole("heading", { name: "Archive policy" }).waitFor();
    return;
  }
  throw new Error(`Expected login to land on a protected route, got ${currentPath}`);
}

async function signOut(page) {
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await page.waitForURL(/\/login$/);
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();
}

async function changePasswordViaSettings(page, currentPassword, nextPassword) {
  const changePasswordResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/settings/password") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.locator("#current-password").fill(currentPassword);
  await page.locator("#new-password").fill(nextPassword);
  await page.locator("#confirm-password").fill(nextPassword);
  await page.getByRole("button", { name: "Update password" }).click();
  const response = await changePasswordResponse;
  assertStatus(
    response.status() >= 200 && response.status() < 300,
    `Password update failed with ${response.status()}`
  );
}

async function importBundle(page, bundlePath) {
  const importResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/import") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.locator("#import-file").setInputFiles(bundlePath);
  const response = await importResponse;
  assertStatus(
    response.status() >= 200 && response.status() < 300,
    `Import failed with ${response.status()}`
  );
  await page.getByText("Import finished and replaced the current board data.").waitFor();
}

async function importBundleExpectFailure(page, bundlePath, { expectedMessage, expectedStatus }) {
  const importResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/import") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.locator("#import-file").setInputFiles(bundlePath);
  const response = await importResponse;
  assertStatus(
    response.status() === expectedStatus,
    `Expected failed import status ${expectedStatus}, got ${response.status()}`
  );
  await page.getByText(expectedMessage).waitFor();
}

function buildInvalidBundle(bundle) {
  const nextBundle = JSON.parse(JSON.stringify(bundle));
  nextBundle.version = " ";
  nextBundle.exportedAt = 0;
  return nextBundle;
}

function findExtraSession(previousSessions, nextSessions) {
  const previousTokens = new Set(previousSessions.map((session) => session.token));
  const extras = nextSessions.filter((session) => !previousTokens.has(session.token));
  assertStatus(extras.length === 1, `Expected exactly one new session, got ${JSON.stringify(nextSessions)}`);
  return extras[0];
}

async function locateSavedSessionRow(page, session) {
  const [lastSeenText, expiresText] = await Promise.all([
    formatBrowserDate(page, session.lastSeenAt),
    formatBrowserDate(page, session.expiresAt),
  ]);
  const row = page
    .locator(".archive-item")
    .filter({ hasText: "Saved session" })
    .filter({ hasText: `Last seen ${lastSeenText}` })
    .filter({ hasText: `expires ${expiresText}` })
    .filter({ hasText: `Client ${session.clientIP || "unknown"}` })
    .first();
  await row.waitFor({ state: "visible", timeout: 10000 });
  return row;
}

async function formatBrowserDate(page, value) {
  return page.evaluate((timestamp) => new Date(timestamp).toLocaleString(), value);
}

function buildMutatedBundle(bundle, archiveRetentionDays) {
  const nextBundle = JSON.parse(JSON.stringify(bundle));
  nextBundle.settings = {
    ...nextBundle.settings,
    archiveRetentionDays,
  };

  const queuedSortOrders = (nextBundle.tasks ?? [])
    .filter((task) => task.status === "queued")
    .map((task) => task.sort_order ?? 0);
  const nextSortOrder =
    queuedSortOrders.length > 0 ? Math.max(...queuedSortOrders) + 1 : 0;

  nextBundle.tasks = [
    ...(nextBundle.tasks ?? []),
    {
      id: importedTaskID,
      title: importedTaskTitle,
      note: "Imported by the settings smoke lane to verify backup restore coverage.",
      due: "2026-06-30",
      priority: "high",
      status: "queued",
      sort_order: nextSortOrder,
      lastUpdated: Date.now(),
    },
  ];

  return nextBundle;
}

function bundlesMatchForState(left, right) {
  return canonicalizeBundleState(left) === canonicalizeBundleState(right);
}

function canonicalizeBundleState(bundle) {
  return JSON.stringify({
    settings: {
      archiveRetentionDays: bundle?.settings?.archiveRetentionDays ?? null,
    },
    tasks: canonicalizeTasks(bundle?.tasks ?? []),
    archived: canonicalizeArchivedTasks(bundle?.archived ?? []),
  });
}

function canonicalizeTasks(tasks) {
  return [...tasks]
    .map((task) => ({
      due: task.due,
      id: task.id,
      lastUpdated: task.lastUpdated ?? 0,
      note: task.note ?? "",
      priority: task.priority,
      sort_order: task.sort_order ?? task.sortOrder ?? 0,
      status: task.status,
      title: task.title,
    }))
    .sort(compareCanonicalItems);
}

function canonicalizeArchivedTasks(tasks) {
  return [...tasks]
    .map((task) => ({
      archivedAt: task.archivedAt ?? 0,
      due: task.due,
      id: task.id,
      note: task.note ?? "",
      priority: task.priority,
      sort_order: task.sort_order ?? task.sortOrder ?? 0,
      status: task.status,
      title: task.title,
    }))
    .sort(compareCanonicalItems);
}

function compareCanonicalItems(left, right) {
  return JSON.stringify(left).localeCompare(JSON.stringify(right));
}

function selectRetentionDays(exclusions) {
  const candidates = [21, 45, 60, 90, 120];
  const excludedValues = new Set(exclusions.filter((value) => value != null));
  const candidate = candidates.find((value) => !excludedValues.has(value));
  assertStatus(candidate != null, "Unable to find a unique archive retention value for smoke.");
  return candidate;
}

async function requestJson(page, url, init = {}) {
  return page.evaluate(
    async ({ url, init, requestTimeoutMs }) => {
      const controller = new AbortController();
      const timeout = setTimeout(
        () => controller.abort(`request timed out after ${requestTimeoutMs}ms`),
        requestTimeoutMs
      );
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
