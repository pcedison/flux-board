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
const requestTimeoutMs = parseInteger(process.env.REQUEST_TIMEOUT_MS, browserName === "webkit" ? 30000 : 15000);
const smokeTimeoutMs = parseInteger(process.env.SMOKE_TIMEOUT_MS, 120000);
const desktopViewport = { width: 1440, height: 960 };
const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const runSeed = `${Date.now()}-${process.pid}`;
const resultsDir =
  process.env.TEST_RESULTS_DIR ??
  path.join("test-results", "e2e", `board-dnd-smoke-${timestamp}`);

if (!password) {
  console.error("Missing FLUX_PASSWORD (or APP_PASSWORD) env var.");
  process.exit(1);
}

await mkdir(resultsDir, { recursive: true });

const overallTimeout = setTimeout(() => {
  console.error(`DnD smoke timed out after ${smokeTimeoutMs}ms.`);
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
  await page.getByRole("button", { name: "Sign in" }).click();
  const loginResult = await loginResponse;
  if (loginResult.status() !== 200) {
    throw new Error(`Login failed with ${loginResult.status()}`);
  }
  await page.waitForURL(/\/board$/);
  await page.getByRole("heading", { name: "New task" }).waitFor();

  logStep("Create tasks");
  const firstTaskTitle = `DnD smoke first ${runSeed}`;
  const secondTaskTitle = `${firstTaskTitle} next`;

  await createQueuedTask(page, {
    due: "2026-05-01",
    note: "First queued task for same-lane drag smoke.",
    title: firstTaskTitle,
  });
  await createQueuedTask(page, {
    due: "2026-05-02",
    note: "Second queued task for same-lane drag smoke.",
    title: secondTaskTitle,
  });

  const queuedLane = page.locator('section.lane[aria-labelledby="lane-queued"]');
  await queuedLane.waitFor({ state: "visible" });
  const queuedTaskTitlesBefore = await getQueuedTaskTitles(queuedLane);
  assertStatus(
    queuedTaskTitlesBefore.includes(firstTaskTitle) && queuedTaskTitlesBefore.includes(secondTaskTitle),
    "Queued lane should contain both smoke tasks before dragging."
  );

  const sourceTitle = queuedTaskTitlesBefore.indexOf(firstTaskTitle) < queuedTaskTitlesBefore.indexOf(secondTaskTitle)
    ? firstTaskTitle
    : secondTaskTitle;
  const targetTitle = sourceTitle === firstTaskTitle ? secondTaskTitle : firstTaskTitle;

  const sourceCard = queuedLane.locator("article").filter({ hasText: sourceTitle }).first();
  const targetCard = queuedLane.locator("article").filter({ hasText: targetTitle }).first();

  await sourceCard.scrollIntoViewIfNeeded();
  await targetCard.scrollIntoViewIfNeeded();

  logStep("Drag");
  await dragPointerToTarget(page, sourceCard, targetCard);

  await page.waitForFunction(
    ({ sourceTitle, targetTitle }) => {
      const lane = document.querySelector('section.lane[aria-labelledby="lane-queued"]');
      if (!lane) {
        return false;
      }
      const titles = Array.from(lane.querySelectorAll("article strong"))
        .map((element) => element.textContent?.trim() || "")
        .filter(Boolean);
      return titles.indexOf(sourceTitle) > titles.indexOf(targetTitle);
    },
    { sourceTitle, targetTitle }
  );

  const queuedTaskTitlesAfter = await getQueuedTaskTitles(queuedLane);
  assertStatus(
    queuedTaskTitlesAfter.indexOf(sourceTitle) > queuedTaskTitlesAfter.indexOf(targetTitle),
    "Queued lane DOM order should place the dragged task below the target task."
  );

  await writeFile(
    path.join(resultsDir, "summary.json"),
    JSON.stringify(
      {
        baseURL,
        browser: browserName,
        lane: "queued",
        tasks: [firstTaskTitle, secondTaskTitle],
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
        lane: "queued",
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
  await page.getByRole("button", { name: "Create task" }).click();
  const response = await createResponse;
  assertStatus(response.ok(), `Create task failed with ${response.status()}`);
  await page.locator("article", { hasText: title }).first().waitFor({ state: "visible", timeout: 10000 });
}

async function getQueuedTaskTitles(queuedLane) {
  return (await queuedLane.locator("article strong").allTextContents()).map((title) => title.trim()).filter(Boolean);
}

async function dragPointerToTarget(page, sourceHandle, targetCard) {
  const sourceBox = await sourceHandle.boundingBox();
  assertStatus(sourceBox != null, "Drag source should be visible.");

  const requiresExtendedPointerPath = browserName === "webkit";
  const source = centerPoint(sourceBox);

  await sourceHandle.hover();
  if (browserName === "firefox" || browserName === "webkit") {
    await installTemporaryFirefoxSelectionGuard(page);
  }

  try {
    await page.mouse.move(source.x, source.y);
    await page.mouse.down();

    const targetBox = await targetCard.boundingBox();
    assertStatus(targetBox != null, "Drag target should be visible.");

    const target =
      requiresExtendedPointerPath
        ? {
            x: targetBox.x + targetBox.width / 2,
            y: targetBox.y + targetBox.height - Math.max(12, Math.min(32, targetBox.height / 4)),
          }
        : centerPoint(targetBox);
    const midPoint = {
      x: source.x + Math.max(16, Math.min(48, Math.abs(target.x - source.x) / 3)),
      y: source.y + Math.max(16, Math.min(48, Math.abs(target.y - source.y) / 3)),
    };

    if (requiresExtendedPointerPath) {
      await page.mouse.move(midPoint.x, midPoint.y, { steps: 8 });
      await page.mouse.move(target.x, target.y, { steps: 12 });
      await page.waitForTimeout(75);
    } else {
      await page.mouse.move(midPoint.x, midPoint.y, { steps: 8 });
      await page.mouse.move(target.x, target.y, { steps: 12 });
    }
    await page.mouse.up();
  } finally {
    if (browserName === "firefox" || browserName === "webkit") {
      await removeTemporaryFirefoxSelectionGuard(page);
    }
  }
}

async function installTemporaryFirefoxSelectionGuard(page) {
  await page.evaluate(() => {
    if (document.getElementById("smoke-firefox-selection-guard")) {
      return;
    }

    const style = document.createElement("style");
    style.id = "smoke-firefox-selection-guard";
    style.textContent = `
      html,
      body,
      .board-grid,
      .board-grid * {
        user-select: none !important;
        -moz-user-select: none !important;
      }
    `;
    document.head.append(style);
  });
}

async function removeTemporaryFirefoxSelectionGuard(page) {
  await page.evaluate(() => {
    document.getElementById("smoke-firefox-selection-guard")?.remove();
  });
}

function centerPoint(box) {
  return {
    x: box.x + box.width / 2,
    y: box.y + box.height / 2,
  };
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
