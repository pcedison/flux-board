import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { BoardSnapshotPage } from "./BoardSnapshotPage";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

vi.mock("../lib/useBoardSnapshot", () => ({
  useBoardSnapshot: vi.fn(),
}));

const mockedUseBoardSnapshot = vi.mocked(useBoardSnapshot);

describe("BoardSnapshotPage", () => {
  beforeEach(() => {
    mockedUseBoardSnapshot.mockReset();
  });

  it("renders an error panel when the snapshot fails", () => {
    mockedUseBoardSnapshot.mockReturnValue({
      data: undefined,
      error: new Error("api down"),
      isPending: false,
    } as ReturnType<typeof useBoardSnapshot>);

    render(<BoardSnapshotPage />);

    expect(screen.getByRole("alert")).toHaveTextContent("Failed to load board snapshot");
    expect(screen.getByRole("alert")).toHaveTextContent("api down");
  });

  it("renders lane cards and archive totals from the snapshot", () => {
    mockedUseBoardSnapshot.mockReturnValue({
      data: {
        session: { authenticated: true, expiresAt: 1 },
        tasks: [
          {
            id: "a",
            title: "Queue me",
            note: "first lane",
            due: "2026-04-20",
            priority: "medium",
            status: "queued",
            sort_order: 0,
            lastUpdated: 1,
          },
          {
            id: "b",
            title: "Do me",
            note: "",
            due: "2026-04-21",
            priority: "high",
            status: "active",
            sort_order: 1,
            lastUpdated: 2,
          },
        ],
        archived: [
          {
            id: "c",
            title: "Archived",
            note: "",
            due: "2026-04-22",
            priority: "critical",
            status: "done",
            sort_order: 0,
            archivedAt: 3,
          },
        ],
      },
      error: null,
      isPending: false,
    } as ReturnType<typeof useBoardSnapshot>);

    render(<BoardSnapshotPage />);

    expect(screen.getByText("Queue me")).toBeInTheDocument();
    expect(screen.getByText("Do me")).toBeInTheDocument();
    expect(screen.getByText("Due 2026-04-20 / order 0")).toBeInTheDocument();
    expect(screen.getByText("1 archived cards")).toBeInTheDocument();
  });
});
