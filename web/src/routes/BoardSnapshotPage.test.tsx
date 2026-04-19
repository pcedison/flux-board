import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { BoardSnapshotPage } from "./BoardSnapshotPage";
import { ApiError, archiveTask, createTask, restoreTask } from "../lib/api";
import { authSessionQueryKey } from "../lib/useAuthSession";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

vi.mock("../lib/useBoardSnapshot", async () => {
  const actual = await vi.importActual("../lib/useBoardSnapshot");
  return {
    ...actual,
    useBoardSnapshot: vi.fn(),
  };
});

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    archiveTask: vi.fn(),
    createTask: vi.fn(),
    restoreTask: vi.fn(),
  };
});

const mockedUseBoardSnapshot = vi.mocked(useBoardSnapshot);
const mockedCreateTask = vi.mocked(createTask);
const mockedArchiveTask = vi.mocked(archiveTask);
const mockedRestoreTask = vi.mocked(restoreTask);

describe("BoardSnapshotPage", () => {
  beforeEach(() => {
    mockedUseBoardSnapshot.mockReset();
    mockedCreateTask.mockReset().mockResolvedValue({
      id: "new-task",
      title: "Created",
      note: "",
      due: "2026-05-01",
      priority: "medium",
      status: "queued",
      sort_order: 0,
      lastUpdated: 10,
    });
    mockedArchiveTask.mockReset().mockResolvedValue({
      id: "a",
      title: "Queue me",
      note: "",
      due: "2026-04-20",
      priority: "medium",
      status: "queued",
      sort_order: 0,
      archivedAt: 10,
    });
    mockedRestoreTask.mockReset().mockResolvedValue({
      id: "c",
      title: "Archived",
      note: "",
      due: "2026-04-22",
      priority: "critical",
      status: "done",
      sort_order: 0,
      lastUpdated: 11,
    });
  });

  it("renders an error panel when the snapshot fails", () => {
    mockedUseBoardSnapshot.mockReturnValue({
      data: undefined,
      error: new Error("api down"),
      isPending: false,
    } as ReturnType<typeof useBoardSnapshot>);

    renderPage();

    expect(screen.getByRole("alert")).toHaveTextContent("Unable to load the board");
    expect(screen.getByRole("alert")).toHaveTextContent("api down");
  });

  it("renders collapsed lanes, create controls, and archive totals from the snapshot", () => {
    mockSnapshot({
      tasks: [
        buildTask("a", "Queue me", "queued", 0, 1),
        buildTask("d", "Queue next", "queued", 1, 4),
        buildTask("b", "Do me", "active", 0, 2),
      ],
    });

    renderPage();

    const queuedLane = screen.getByRole("region", { name: "Queued" });
    expect(within(queuedLane).getByRole("button", { name: /2 tasks hidden/i })).toBeInTheDocument();
    expect(within(queuedLane).queryByText("Queue me")).not.toBeInTheDocument();

    fireEvent.click(within(queuedLane).getByRole("button", { name: /2 tasks hidden/i }));

    expect(screen.getByText("Queue me")).toBeInTheDocument();
    expect(screen.getByText("Queue next")).toBeInTheDocument();
    expect(screen.getAllByText("Due 2026-04-20").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Create task" })).toBeInTheDocument();
    expect(screen.getByText("1 archived task")).toBeInTheDocument();
  });

  it("has no accessibility violations for the default board snapshot", async () => {
    mockSnapshot({
      tasks: [
        buildTask("a", "Queue me", "queued", 0, 1),
        buildTask("d", "Queue next", "queued", 1, 4),
      ],
    });

    const { container } = renderPageView();
    fireEvent.click(screen.getByRole("button", { name: /2 tasks hidden/i }));

    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });

  it("shows inline field validation before create and focuses the missing field", () => {
    mockSnapshot();

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Create task" }));

    expect(screen.getByText("Enter a task title before creating a card.")).toBeInTheDocument();
    expect(screen.getByText("Choose a due date before creating a card.")).toBeInTheDocument();
    expect(screen.getByLabelText("Title")).toHaveFocus();
    expect(mockedCreateTask).not.toHaveBeenCalled();
  });

  it("submits the create task form through the typed API helper", async () => {
    mockSnapshot();

    renderPage();

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Ship mutations" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-04-30" } });
    fireEvent.change(screen.getByLabelText("Note"), { target: { value: "Keep it read-write, not drag-first." } });
    fireEvent.click(screen.getByRole("button", { name: "Create task" }));

    await waitFor(() =>
      expect(mockedCreateTask).toHaveBeenCalledWith({
        title: "Ship mutations",
        due: "2026-04-30",
        note: "Keep it read-write, not drag-first.",
        priority: "medium",
      }),
    );
  });

  it("opens task details when a card is selected and archives from the side panel", async () => {
    mockSnapshot();

    renderPage();

    fireEvent.click(screen.getByText("Queue me").closest("article")!);

    expect(screen.getByRole("heading", { name: "Task details" })).toBeInTheDocument();
    expect(screen.getByDisplayValue("Queue me")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Archive task" }));

    await waitFor(() => expect(mockedArchiveTask).toHaveBeenCalledWith("a"));
  });

  it("restores archived tasks from the archive panel", async () => {
    mockSnapshot();

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Restore Archived" }));

    await waitFor(() => expect(mockedRestoreTask).toHaveBeenCalledWith("c"));
  });

  it("clears auth-session ownership when a mutation returns 401", async () => {
    mockSnapshot();
    mockedArchiveTask.mockRejectedValueOnce(new ApiError("unauthorized", 401));

    const queryClient = renderPage({
      authSession: { authenticated: true, expiresAt: 1 },
    });

    fireEvent.click(screen.getByText("Queue me").closest("article")!);
    fireEvent.click(screen.getByRole("button", { name: "Archive task" }));

    await waitFor(() => expect(queryClient.getQueryData(authSessionQueryKey)).toBeNull());
  });
});

function renderPage(options?: { authSession?: { authenticated: boolean; expiresAt: number } | null }) {
  return renderPageView(options).queryClient;
}

function renderPageView(options?: { authSession?: { authenticated: boolean; expiresAt: number } | null }) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  if (options && Object.prototype.hasOwnProperty.call(options, "authSession")) {
    queryClient.setQueryData(authSessionQueryKey, options.authSession ?? null);
  }

  const view = render(
    <QueryClientProvider client={queryClient}>
      <BoardSnapshotPage />
    </QueryClientProvider>,
  );

  return { queryClient, ...view };
}

function mockSnapshot(overrides?: Partial<NonNullable<ReturnType<typeof useBoardSnapshot>["data"]>>) {
  const data: NonNullable<ReturnType<typeof useBoardSnapshot>["data"]> = {
    session: { authenticated: true, expiresAt: 1 },
    tasks: [
      buildTask("a", "Queue me", "queued", 0, 1, "first lane"),
      buildTask("b", "Do me", "active", 1, 2),
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
    ...overrides,
  };

  mockedUseBoardSnapshot.mockReturnValue({
    data,
    error: null,
    isPending: false,
  } as ReturnType<typeof useBoardSnapshot>);
}

function buildTask(
  id: string,
  title: string,
  status: "queued" | "active" | "done",
  sortOrder: number,
  lastUpdated: number,
  note = "",
) {
  return {
    id,
    title,
    note,
    due: "2026-04-20",
    priority: status === "active" ? "high" : "medium",
    status,
    sort_order: sortOrder,
    lastUpdated,
  } as const;
}
