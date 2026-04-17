import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { OverviewPage } from "./OverviewPage";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

vi.mock("../lib/useBoardSnapshot", () => ({
  useBoardSnapshot: vi.fn(),
}));

const mockedUseBoardSnapshot = vi.mocked(useBoardSnapshot);

describe("OverviewPage", () => {
  beforeEach(() => {
    mockedUseBoardSnapshot.mockReset();
  });

  it("renders loading state while the snapshot is pending", async () => {
    mockedUseBoardSnapshot.mockReturnValue({
      data: undefined,
      error: null,
      isPending: true,
    } as ReturnType<typeof useBoardSnapshot>);

    const { container } = render(
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>,
    );

    expect(screen.getByText("Loading")).toBeInTheDocument();
    expect(screen.getByText("Reading the current auth and board snapshot from the Go API.")).toBeInTheDocument();
    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });

  it("renders authenticated totals from the snapshot", async () => {
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
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>,
    );

    expect(screen.getByText("Authenticated session detected")).toBeInTheDocument();
    expect(screen.getByText("Queued")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Done")).toBeInTheDocument();
    expect(screen.getByText("Archived")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open board route" })).toHaveAttribute("href", "/board");
    expect(screen.getAllByText("1")).toHaveLength(4);
    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });
});
