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
  path.join("test-results", "e2e", `board-keyboard-smoke-${timestamp}`);

if (!password) {
  console.error("Missing FLUX_PASSWORD (or APP_PASSWORD) env var.");
  process.exit(1);
}

await mkdir(resultsDir, { recursive: true });

const overallTimeout = setTimeout(() => {
  console.error(`Board keyboard smoke timed out after ${smokeTimeoutMs}ms.`);
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
  logStep("Login");
  await page.goto(`${baseURL}/login`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Sign in to view the board" }).waitFor();

  const loginResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/auth/login") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.getByLabel("Password").fill(password);
  await page.keyboard.press("Tab");
  await assertFocused(page, page.getByRole("button", { name: "Sign in" }), "Sign in button");
  await page.keyboard.press("Enter");
  const loginResult = await loginResponse;
  assertStatus(loginResult.status() === 200, `Login failed with ${loginResult.status()}`);

  await page.waitForURL(/\/board$/);
  await page.getByRole("heading", { name: "New task" }).waitFor();
  await page.getByLabel("Title").waitFor({ state: "visible" });

  logStep("Create tasks");
  const taskSeed = Date.now();
  const firstTaskTitle = `Keyboard smoke ${taskSeed} one`;
  const secondTaskTitle = `Keyboard smoke ${taskSeed} two`;
  const thirdTaskTitle = `Keyboard smoke ${taskSeed} three`;
  const firstTaskNote = "Keyboard smoke queued task one.";
  const secondTaskNote = "Keyboard smoke queued task two.";
  const thirdTaskNote = "Keyboard smoke queued task three.";

  await createQueuedTask(page, {
    due: "2026-05-04",
    note: firstTaskNote,
    title: firstTaskTitle,
  });
  await assertFocused(page, page.getByLabel("Title"), "Title input after first create");

  await createQueuedTask(page, {
    due: "2026-05-05",
    note: secondTaskNote,
    title: secondTaskTitle,
  });
  await assertFocused(page, page.getByLabel("Title"), "Title input after second create");

  await createQueuedTask(page, {
    due: "2026-05-06",
    note: thirdTaskNote,
    title: thirdTaskTitle,
  });
  await assertFocused(page, page.getByLabel("Title"), "Title input after third create");

  const tasksResponse = await requestJson(page, "/api/tasks");
  assertStatus(tasksResponse.status === 200, `Expected /api/tasks to return 200, got ${tasksResponse.status}`);
  const firstTask = (tasksResponse.body?.tasks ?? []).find((task) => task.title === firstTaskTitle);
  assertStatus(firstTask != null, `Expected to find ${firstTaskTitle} in /api/tasks.`);

  const moveResult = await requestJson(page, `/api/tasks/${firstTask.id}/reorder`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status: "active" }),
  });
  assertStatus(moveResult.status === 200, `Move to active failed with ${moveResult.status}`);
  await page.goto(`${baseURL}/board`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "New task" }).waitFor();
  const activeLane = page.locator('section.lane[aria-labelledby="lane-active"]');
  const movedCard = activeLane.locator("article").filter({ hasText: firstTaskTitle }).first();
  await movedCard.waitFor({ state: "visible", timeout: 10000 });

  const queuedLane = page.locator('section.lane[aria-labelledby="lane-queued"]');
  const secondCard = queuedLane.locator("article").filter({ hasText: secondTaskTitle }).first();
  const thirdCard = queuedLane.locator("article").filter({ hasText: thirdTaskTitle }).first();
  const tabbableQueuedCard = queuedLane.locator("article[tabindex='0']").first();
  await queuedLane.waitFor({ state: "visible" });
  await secondCard.waitFor({ state: "visible", timeout: 10000 });
  await thirdCard.waitFor({ state: "visible", timeout: 10000 });
  await tabbableQueuedCard.waitFor({ state: "visible", timeout: 10000 });

  await secondCard.scrollIntoViewIfNeeded();
  await thirdCard.scrollIntoViewIfNeeded();
  await movedCard.scrollIntoViewIfNeeded();
  await secondCard.scrollIntoViewIfNeeded();
  assertStatus(
    await tabbableQueuedCard.evaluate((element) => element.tabIndex === 0),
    "Queued lane should expose one tabbable card shell."
  );

  logStep("Keyboard navigation");
  await secondCard.focus();
  await assertFocused(page, secondCard, `Queued card shell for ${secondTaskTitle}`);
  await waitForTabIndex(page, secondCard, 0, `${secondTaskTitle} card shell should become the active roving target after focus.`);
  await assertCardShellFocused(page, "queued", secondTaskTitle);

  await page.keyboard.press("ArrowDown");
  await assertFocused(page, thirdCard, `Queued card shell for ${thirdTaskTitle}`);
  await assertCardShellFocused(page, "queued", thirdTaskTitle);

  await page.keyboard.press("ArrowUp");
  await assertFocused(page, secondCard, `Queued card shell for ${secondTaskTitle}`);
  await assertCardShellFocused(page, "queued", secondTaskTitle);

  await page.keyboard.press("ArrowRight");
  await assertFocused(page, movedCard, `Moved card shell for ${firstTaskTitle} in the active lane`);
  await assertCardShellFocused(page, "active", firstTaskTitle);

  await page.keyboard.press("ArrowLeft");
  const leftLaneFocus = await getFocusedCardInfo(page);
  assertStatus(leftLaneFocus?.laneStatus === "queued", "ArrowLeft from the active lane should move focus into the queued lane.");
  assertStatus(leftLaneFocus?.title === secondTaskTitle, `Expected ArrowLeft to restore focus to ${secondTaskTitle}.`);

  await page.keyboard.press("ArrowRight");
  const rightLaneFocus = await getFocusedCardInfo(page);
  assertStatus(rightLaneFocus?.laneStatus === "active", "ArrowRight from the queued lane should move focus back into the active lane.");
  assertStatus(rightLaneFocus?.title === firstTaskTitle, `Expected ArrowRight to focus ${firstTaskTitle} in the active lane.`);

  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        tasks: [firstTaskTitle, secondTaskTitle, thirdTaskTitle],
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

async function createQueuedTask(page, { title, due, note }) {
  await page.getByLabel("Title").fill(title);
  await page.getByLabel("Due date").fill(due);
  await page.getByLabel("Note").fill(note);
  const createResponse = page.waitForResponse(
    (res) => res.url().endsWith("/api/tasks") && res.request().method() === "POST",
    { timeout: 10000 }
  );
  await page.keyboard.press("Tab");
  await assertFocused(page, page.getByRole("button", { name: "Create task" }), "Create task button");
  await page.keyboard.press("Enter");
  const response = await createResponse;
  assertStatus(response.ok(), `Create task failed with ${response.status()}`);
  await page.locator("article", { hasText: title }).first().waitFor({ state: "visible", timeout: 10000 });
}

async function assertCardShellFocused(page, laneStatus, title) {
  assertStatus(
    await page.evaluate(
      ({ laneStatus, title }) => {
        const active = document.activeElement;
        if (!active || active.tagName !== "ARTICLE") {
          return false;
        }
        const lane = active.closest(`section.lane[aria-labelledby="lane-${laneStatus}"]`);
        return Boolean(lane) && String(active.textContent || "").includes(title);
      },
      { laneStatus, title }
    ),
    `Expected the ${laneStatus} card shell for ${title} to be focused.`
  );
}

async function getFocusedCardInfo(page) {
  return page.evaluate(() => {
    const active = document.activeElement;
    if (!active || active.tagName !== "ARTICLE") {
      return null;
    }

    const lane = active.closest("section.lane");
    const laneLabelledBy = lane?.getAttribute("aria-labelledby") ?? "";
    const laneStatus = laneLabelledBy.startsWith("lane-") ? laneLabelledBy.slice("lane-".length) : null;
    const title = active.querySelector("strong")?.textContent?.trim() ?? "";

    return { laneStatus, title };
  });
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

async function waitForTabIndex(page, locator, expected, label) {
  const handle = await locator.elementHandle();
  assertStatus(handle != null, `${label} (missing element)`);
  await page.waitForFunction(
    ({ element, expected }) => element.tabIndex === expected,
    { element: handle, expected },
    { timeout: 10000 }
  );
}

async function tabUntilFocused(page, locator, maxTabs, label) {
  for (let count = 0; count < maxTabs; count += 1) {
    if (await isFocused(locator)) {
      return;
    }
    await page.keyboard.press("Tab");
  }

  assertStatus(await isFocused(locator), `${label} should be reachable by tabbing within the card actions.`);
}

async function assertFocused(page, locator, label) {
  const handle = await locator.elementHandle();
  assertStatus(handle != null, `${label} should exist.`);
  await page.waitForFunction((element) => document.activeElement === element, handle, { timeout: 10000 });
}

async function isFocused(locator) {
  const handle = await locator.elementHandle();
  if (!handle) {
    return false;
  }
  return handle.evaluate((element) => document.activeElement === element);
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
