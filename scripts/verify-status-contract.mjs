import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const baseURL = normalizeBaseUrl(process.env.BASE_URL ?? "http://127.0.0.1:8080");
const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const resultsDir =
  process.env.TEST_RESULTS_DIR ??
  path.join("test-results", "status-contract", `verify-status-contract-${timestamp}`);
const timeoutMs = parseInteger(process.env.STATUS_TIMEOUT_MS, 15000);
const expectStatus = parseExpected(
  process.env.EXPECT_STATUS ?? "ready",
  new Set(["ready", "degraded", "any"]),
  "EXPECT_STATUS"
);
const expectNeedsSetup = parseExpected(
  process.env.EXPECT_NEEDS_SETUP ?? "any",
  new Set(["true", "false", "any"]),
  "EXPECT_NEEDS_SETUP"
);
const expectEnvironment = normalizeOptional(process.env.EXPECT_ENVIRONMENT);
const expectVersion = normalizeOptional(process.env.EXPECT_VERSION);
const requireMetrics = parseBoolean(process.env.REQUIRE_METRICS, true);

await mkdir(resultsDir, { recursive: true });

const summary = {
  baseURL,
  resultsDir,
  expected: {
    status: expectStatus,
    needsSetup: expectNeedsSetup,
    environment: expectEnvironment ?? "any",
    version: expectVersion ?? "any",
    requireMetrics,
  },
  routes: {},
  status: "failed",
};

try {
  const healthz = await requestRoute("/healthz");
  await writeJSONArtifact("healthz.json", healthz);
  summary.routes.healthz = summarizeResponse(healthz);
  assert(healthz.status === 200, `Expected /healthz to return 200, got ${healthz.status}`);
  assert(healthz.json?.status === "ok", `Expected /healthz body.status=ok, got ${JSON.stringify(healthz.json)}`);

  const readyz = await requestRoute("/readyz");
  await writeJSONArtifact("readyz.json", readyz);
  summary.routes.readyz = summarizeResponse(readyz);
  assert(readyz.status === 200, `Expected /readyz to return 200, got ${readyz.status}`);
  assert(readyz.json?.status === "ready", `Expected /readyz body.status=ready, got ${JSON.stringify(readyz.json)}`);

  const appStatus = await requestRoute("/api/status");
  await writeJSONArtifact("status.json", appStatus);
  summary.routes.apiStatus = summarizeResponse(appStatus);
  assert(
    appStatus.status === 200 || appStatus.status === 503,
    `Expected /api/status to return 200 or 503, got ${appStatus.status}`
  );
  assert(isObject(appStatus.json), `Expected /api/status to return a JSON object, got ${JSON.stringify(appStatus.json)}`);

  const statusBody = appStatus.json;
  assert(statusBody.status === "ready" || statusBody.status === "degraded", `Unexpected app status ${statusBody.status}`);
  if (expectStatus !== "any") {
    assert(statusBody.status === expectStatus, `Expected app status ${expectStatus}, got ${statusBody.status}`);
    const expectedHTTPStatus = expectStatus === "ready" ? 200 : 503;
    assert(appStatus.status === expectedHTTPStatus, `Expected /api/status HTTP ${expectedHTTPStatus}, got ${appStatus.status}`);
  }

  if (expectNeedsSetup !== "any") {
    assert(
      String(statusBody.needsSetup) === expectNeedsSetup,
      `Expected needsSetup=${expectNeedsSetup}, got ${JSON.stringify(statusBody.needsSetup)}`
    );
  }
  if (expectEnvironment !== null) {
    assert(statusBody.environment === expectEnvironment, `Expected environment=${expectEnvironment}, got ${statusBody.environment}`);
  }
  if (expectVersion !== null) {
    assert(statusBody.version === expectVersion, `Expected version=${expectVersion}, got ${statusBody.version}`);
  }

  assert(typeof statusBody.version === "string" && statusBody.version.length > 0, "Expected non-empty status.version");
  assert(
    typeof statusBody.runtimeArtifact === "string" && statusBody.runtimeArtifact.length > 0,
    "Expected non-empty status.runtimeArtifact"
  );
  assert(statusBody.runtimeOwnershipPath === "/", `Expected runtimeOwnershipPath=/, got ${statusBody.runtimeOwnershipPath}`);
  assert(statusBody.legacyRollbackPath === "/legacy/", `Expected legacyRollbackPath=/legacy/, got ${statusBody.legacyRollbackPath}`);
  assert(typeof statusBody.generatedAt === "number" && statusBody.generatedAt > 0, `Expected generatedAt timestamp, got ${statusBody.generatedAt}`);
  assert(typeof statusBody.archiveCleanupEvery === "string", "Expected archiveCleanupEvery string");
  assert(typeof statusBody.sessionCleanupEvery === "string", "Expected sessionCleanupEvery string");
  assert(Array.isArray(statusBody.checks), "Expected status.checks array");

  const checkNames = new Set(statusBody.checks.map((check) => check?.name));
  for (const required of ["database", "bootstrap", "archive-retention"]) {
    assert(checkNames.has(required), `Expected /api/status checks to include ${required}, got ${JSON.stringify(statusBody.checks)}`);
  }

  const statusPage = await requestRoute("/status", { expectJSON: false });
  await writeTextArtifact("status-page.html", statusPage.text);
  summary.routes.statusPage = summarizeResponse(statusPage);
  assert(statusPage.status === 200, `Expected /status to return 200, got ${statusPage.status}`);
  assert(statusPage.contentType.includes("text/html"), `Expected /status content-type text/html, got ${statusPage.contentType}`);

  if (requireMetrics) {
    const metrics = await requestRoute("/metrics", { expectJSON: false });
    await writeTextArtifact("metrics.txt", metrics.text);
    summary.routes.metrics = summarizeResponse(metrics);
    assert(metrics.status === 200, `Expected /metrics to return 200, got ${metrics.status}`);
    assert(
      metrics.text.includes("flux_board_http_requests_total") &&
        metrics.text.includes("flux_board_http_request_duration_seconds"),
      "Expected Flux Board HTTP metrics in /metrics output"
    );
  }

  summary.status = "passed";
  console.log(`Status contract verification completed successfully. Results: ${resultsDir}`);
} catch (error) {
  const message = error instanceof Error ? error.message : String(error);
  summary.error = message;
  console.error(message);
  process.exitCode = 1;
} finally {
  await writeFile(path.join(resultsDir, "summary.json"), JSON.stringify(summary, null, 2), "utf8");
}

async function requestRoute(routePath, options = {}) {
  const expectJSON = options.expectJSON !== false;
  const url = new URL(routePath, `${baseURL}/`);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(`request timed out after ${timeoutMs}ms`), timeoutMs);

  try {
    const response = await fetch(url, {
      method: "GET",
      redirect: "follow",
      signal: controller.signal,
    });
    const contentType = response.headers.get("content-type") ?? "";
    const text = await response.text();
    let json = null;
    if (expectJSON || contentType.includes("application/json")) {
      try {
        json = text ? JSON.parse(text) : null;
      } catch {
        json = null;
      }
    }
    return {
      url: url.toString(),
      status: response.status,
      ok: response.ok,
      contentType,
      text,
      json,
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      url: url.toString(),
      status: 0,
      ok: false,
      contentType: "",
      text: "",
      json: { error: message },
    };
  } finally {
    clearTimeout(timeout);
  }
}

function summarizeResponse(response) {
  return {
    url: response.url,
    httpStatus: response.status,
    contentType: response.contentType,
  };
}

async function writeJSONArtifact(filename, response) {
  await writeFile(
    path.join(resultsDir, filename),
    JSON.stringify(
      {
        url: response.url,
        httpStatus: response.status,
        contentType: response.contentType,
        body: response.json,
      },
      null,
      2
    ),
    "utf8"
  );
}

async function writeTextArtifact(filename, contents) {
  await writeFile(path.join(resultsDir, filename), contents, "utf8");
}

function normalizeBaseUrl(value) {
  const url = new URL(value);
  return url.toString().replace(/\/$/, "");
}

function normalizeOptional(value) {
  if (value === undefined) {
    return null;
  }
  const normalized = String(value).trim();
  if (!normalized || normalized.toLowerCase() === "any") {
    return null;
  }
  return normalized;
}

function parseExpected(value, allowed, name) {
  const normalized = String(value).trim().toLowerCase();
  if (!allowed.has(normalized)) {
    throw new Error(`${name} must be one of ${Array.from(allowed).join(", ")}, got ${value}`);
  }
  return normalized;
}

function parseBoolean(value, fallback) {
  if (value === undefined) {
    return fallback;
  }
  const normalized = String(value).trim().toLowerCase();
  if (normalized === "1" || normalized === "true" || normalized === "yes") {
    return true;
  }
  if (normalized === "0" || normalized === "false" || normalized === "no") {
    return false;
  }
  throw new Error(`Expected boolean env var, got ${value}`);
}

function parseInteger(value, fallback) {
  if (value === undefined) {
    return fallback;
  }
  const parsed = Number.parseInt(String(value), 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    throw new Error(`Expected positive integer env var, got ${value}`);
  }
  return parsed;
}

function isObject(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}
