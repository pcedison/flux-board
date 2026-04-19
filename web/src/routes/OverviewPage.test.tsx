import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { OverviewPage } from "./OverviewPage";
import { PreferencesProvider } from "../lib/preferences";
import { useAppStatus } from "../lib/useAppStatus";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

vi.mock("../lib/useBoardSnapshot", () => ({
  useBoardSnapshot: vi.fn(),
}));

vi.mock("../lib/useAppStatus", () => ({
  useAppStatus: vi.fn(),
}));

const mockedUseBoardSnapshot = vi.mocked(useBoardSnapshot);
const mockedUseAppStatus = vi.mocked(useAppStatus);

describe("OverviewPage", () => {
  beforeEach(() => {
    mockedUseBoardSnapshot.mockReset();
    mockedUseAppStatus.mockReset();
  });

  it("renders loading state while the snapshot is pending", async () => {
    mockedUseAppStatus.mockReturnValue({
      data: undefined,
      error: null,
      isPending: true,
    } as ReturnType<typeof useAppStatus>);
    mockedUseBoardSnapshot.mockReturnValue({
      data: undefined,
      error: null,
      isPending: true,
    } as ReturnType<typeof useBoardSnapshot>);

    const { container } = render(
      <PreferencesProvider>
        <MemoryRouter>
          <OverviewPage />
        </MemoryRouter>
      </PreferencesProvider>,
    );

    expect(screen.getByText("Loading")).toBeInTheDocument();
    expect(screen.getByText("Loading your board summary and app status.")).toBeInTheDocument();
    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });

  it("renders authenticated totals from the snapshot", async () => {
    mockedUseAppStatus.mockReturnValue({
      data: {
        status: "ready",
        version: "dev",
        environment: "development",
        needsSetup: false,
        archiveRetentionDays: null,
        runtimeArtifact: "self-contained-root-runtime",
        runtimeOwnershipPath: "/",
        legacyRollbackPath: "/legacy/",
        archiveCleanupEvery: "1h0m0s",
        sessionCleanupEvery: "15m0s",
        generatedAt: new Date("2026-04-30T08:00:00Z").getTime(),
        checks: [
          { name: "database", ok: true, message: "database reachable" },
          { name: "bootstrap", ok: true, message: "admin password already configured" },
        ],
      },
      error: null,
      isPending: false,
    } as ReturnType<typeof useAppStatus>);
    mockedUseBoardSnapshot.mockReturnValue({
      data: {
        session: {
          authenticated: true,
          expiresAt: new Date("2026-04-30T08:30:00Z").getTime(),
        },
        tasks: [
          { id: "a", title: "Queue me", note: "", due: "2026-04-20", priority: "medium", status: "queued", sort_order: 0, lastUpdated: 1 },
          { id: "b", title: "Ship me", note: "", due: "2026-04-21", priority: "high", status: "active", sort_order: 0, lastUpdated: 2 },
          { id: "c", title: "Done me", note: "", due: "2026-04-22", priority: "critical", status: "done", sort_order: 0, lastUpdated: 3 },
        ],
        archived: [
          { id: "d", title: "Archived", note: "", due: "2026-04-23", priority: "medium", status: "done", sort_order: 0, archivedAt: 4 },
        ],
      },
      error: null,
      isPending: false,
    } as ReturnType<typeof useBoardSnapshot>);

    const { container } = render(
      <PreferencesProvider>
        <MemoryRouter>
          <OverviewPage />
        </MemoryRouter>
      </PreferencesProvider>,
    );

    expect(screen.getByText("Ready to use")).toBeInTheDocument();
    expect(screen.getByText("Queued")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Done")).toBeInTheDocument();
    expect(screen.getByText("Archived")).toBeInTheDocument();
    expect(screen.getByText("database reachable")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open board" })).toHaveAttribute("href", "/board");
    expect(screen.getAllByText("1")).toHaveLength(4);
    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });
});
