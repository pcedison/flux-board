import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { BoardSnapshotPage } from "./BoardSnapshotPage";
import { ApiError, archiveTask, createTask, moveTask, restoreTask } from "../lib/api";
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
    moveTask: vi.fn(),
    restoreTask: vi.fn(),
  };
});

const mockedUseBoardSnapshot = vi.mocked(useBoardSnapshot);
const mockedCreateTask = vi.mocked(createTask);
const mockedMoveTask = vi.mocked(moveTask);
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
    mockedMoveTask.mockReset().mockResolvedValue({
      id: "a",
      title: "Queue me",
      note: "",
      due: "2026-04-20",
      priority: "medium",
      status: "active",
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

  it("renders lane cards, create controls, and archive totals from the snapshot", () => {
    mockSnapshot();

    renderPage();

    expect(screen.getByText("Queue me")).toBeInTheDocument();
    expect(screen.getByText("Do me")).toBeInTheDocument();
    expect(screen.getByText("Due 2026-04-20")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Drag Queue me to reorder within Queued" })).toBeInTheDocument();
    expect(screen.getByText("Queue me").closest("article")).toHaveAttribute("tabindex", "0");
    expect(screen.getByRole("button", { name: "Create task" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Archive Queue me" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Restore Archived" })).toBeInTheDocument();
    expect(screen.getByText("1 archived task")).toBeInTheDocument();
  });

  it("has no accessibility violations for the default board snapshot", async () => {
    mockSnapshot();

    renderPage();

    const results = await axe(screen.getByRole("region", { name: "Queued" }));
    expect(results.violations).toHaveLength(0);
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

  it("keeps unrelated controls available while create is pending and restores focus to the title field", async () => {
    mockSnapshot();
    const deferred = createDeferred<Awaited<ReturnType<typeof createTask>>>();
    mockedCreateTask.mockReturnValueOnce(deferred.promise);

    renderPage();

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Ship mutations" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-04-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Create task" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Creating..." })).toBeDisabled());
    expect(screen.getByRole("button", { name: "Move to Active (Queue me)" })).toBeEnabled();

    deferred.resolve({
      id: "new-task",
      title: "Ship mutations",
      note: "",
      due: "2026-04-30",
      priority: "medium",
      status: "queued",
      sort_order: 2,
      lastUpdated: 12,
    });

    await waitFor(() => expect(screen.getByLabelText("Title")).toHaveFocus());
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

  it("triggers explicit move, archive, and restore actions", async () => {
    mockSnapshot();

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Move to Active (Queue me)" }));
    fireEvent.click(screen.getByRole("button", { name: "Archive Queue me" }));
    fireEvent.click(screen.getByRole("button", { name: "Restore Archived" }));

    await waitFor(() =>
      expect(mockedMoveTask).toHaveBeenCalledWith({
        id: "a",
        status: "active",
      }),
    );
    await waitFor(() => expect(mockedArchiveTask).toHaveBeenCalledWith("a"));
    await waitFor(() => expect(mockedRestoreTask).toHaveBeenCalledWith("c"));
  });

  it("surfaces a status message after a successful move action", async () => {
    mockSnapshot();

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Move to Active (Queue me)" }));

    await waitFor(() => expect(screen.getByText("Moved Queue me to Active.")).toBeInTheDocument());
  });

  it("limits move pending state to the active card and restores focus to that card after success", async () => {
    let snapshotData: NonNullable<ReturnType<typeof useBoardSnapshot>["data"]> = {
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
    };
    mockedUseBoardSnapshot.mockImplementation(
      () =>
        ({
          data: snapshotData,
          error: null,
          isPending: false,
        }) as ReturnType<typeof useBoardSnapshot>,
    );
    const deferred = createDeferred<Awaited<ReturnType<typeof moveTask>>>();
    mockedMoveTask.mockReturnValueOnce(deferred.promise);

    const { queryClient, rerender } = renderPageView();

    const moveButton = screen.getByRole("button", { name: "Move to Active (Queue me)" });
    const archiveButton = screen.getByRole("button", { name: "Archive Queue me" });
    const unrelatedButton = screen.getByRole("button", { name: "Back to Queued (Do me)" });

    fireEvent.click(moveButton);

    await waitFor(() => expect(moveButton).toBeDisabled());
    expect(archiveButton).toBeDisabled();
    expect(unrelatedButton).toBeEnabled();

    deferred.resolve({
      id: "a",
      title: "Queue me",
      note: "",
      due: "2026-04-20",
      priority: "medium",
      status: "active",
      sort_order: 0,
      lastUpdated: 10,
    });

    snapshotData = {
      ...snapshotData,
      tasks: [
        {
          id: "a",
          title: "Queue me",
          note: "first lane",
          due: "2026-04-20",
          priority: "medium",
          status: "active",
          sort_order: 0,
          lastUpdated: 10,
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
    };

    rerender(
      <QueryClientProvider client={queryClient}>
        <BoardSnapshotPage />
      </QueryClientProvider>,
    );

    const movedCard = screen.getByText("Queue me").closest("article");
    if (!movedCard) {
      throw new Error("expected moved card article");
    }

    await waitFor(() => expect(movedCard).toHaveFocus());
  });

  it("waits for the refreshed snapshot before restoring focus after a lane move", async () => {
    const deferred = createDeferred<Awaited<ReturnType<typeof moveTask>>>();
    mockedMoveTask.mockReturnValueOnce(deferred.promise);

    let snapshotData: NonNullable<ReturnType<typeof useBoardSnapshot>["data"]> = {
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
      archived: [],
    };

    mockedUseBoardSnapshot.mockImplementation(
      () =>
        ({
          data: snapshotData,
          error: null,
          isPending: false,
        }) as ReturnType<typeof useBoardSnapshot>,
    );

    const { queryClient, rerender } = renderPageView();

    const queuedCard = screen.getByText("Queue me").closest("article");
    if (!queuedCard) {
      throw new Error("expected queued card article");
    }

    fireEvent.click(screen.getByRole("button", { name: "Move to Active (Queue me)" }));

    deferred.resolve({
      id: "a",
      title: "Queue me",
      note: "first lane",
      due: "2026-04-20",
      priority: "medium",
      status: "active",
      sort_order: 0,
      lastUpdated: 10,
    });

    await waitFor(() => expect(screen.getByText("Moved Queue me to Active.")).toBeInTheDocument());
    expect(queuedCard).not.toHaveFocus();

    snapshotData = {
      ...snapshotData,
      tasks: [
        {
          id: "a",
          title: "Queue me",
          note: "first lane",
          due: "2026-04-20",
          priority: "medium",
          status: "active",
          sort_order: 0,
          lastUpdated: 10,
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
    };

    rerender(
      <QueryClientProvider client={queryClient}>
        <BoardSnapshotPage />
      </QueryClientProvider>,
    );

    const movedCard = screen.getByText("Queue me").closest("article");
    if (!movedCard) {
      throw new Error("expected moved card article");
    }

    await waitFor(() => expect(movedCard).toHaveFocus());
    await waitFor(() => expect(movedCard).toHaveAttribute("tabindex", "0"));
  });

  it("uses explicit move-up and move-down controls for lane-local reorder fallback", async () => {
    mockSnapshot({
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
          id: "d",
          title: "Queue next",
          note: "",
          due: "2026-04-23",
          priority: "high",
          status: "queued",
          sort_order: 1,
          lastUpdated: 4,
        },
        {
          id: "e",
          title: "Queue later",
          note: "",
          due: "2026-04-24",
          priority: "critical",
          status: "queued",
          sort_order: 2,
          lastUpdated: 5,
        },
      ],
    });

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Move Queue next up within Queued" }));
    fireEvent.click(screen.getByRole("button", { name: "Move Queue next down within Queued" }));

    await waitFor(() =>
      expect(mockedMoveTask).toHaveBeenNthCalledWith(1, {
        id: "d",
        status: "queued",
        anchorTaskId: "a",
        placeAfter: false,
      }),
    );
    await waitFor(() =>
      expect(mockedMoveTask).toHaveBeenNthCalledWith(2, {
        id: "d",
        status: "queued",
        anchorTaskId: "e",
        placeAfter: true,
      }),
    );
  });

  it("moves focus between card shells with arrow keys", async () => {
    mockSnapshot({
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
          id: "d",
          title: "Queue next",
          note: "",
          due: "2026-04-23",
          priority: "high",
          status: "queued",
          sort_order: 1,
          lastUpdated: 4,
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
        {
          id: "e",
          title: "Done me",
          note: "",
          due: "2026-04-24",
          priority: "critical",
          status: "done",
          sort_order: 2,
          lastUpdated: 5,
        },
      ],
    });

    renderPage();

    const queuedCard = screen.getByText("Queue me").closest("article");
    const nextQueuedCard = screen.getByText("Queue next").closest("article");
    const activeCard = screen.getByText("Do me").closest("article");

    if (!queuedCard || !nextQueuedCard || !activeCard) {
      throw new Error("expected focusable task cards");
    }

    queuedCard.focus();
    fireEvent.keyDown(queuedCard, { key: "ArrowDown" });

    await waitFor(() => expect(nextQueuedCard).toHaveFocus());
    expect(queuedCard).toHaveAttribute("tabindex", "-1");
    expect(nextQueuedCard).toHaveAttribute("tabindex", "0");

    fireEvent.keyDown(nextQueuedCard, { key: "ArrowRight" });

    await waitFor(() => expect(activeCard).toHaveFocus());
    expect(nextQueuedCard).toHaveAttribute("tabindex", "-1");
    expect(activeCard).toHaveAttribute("tabindex", "0");
  });

  it("clears auth-session ownership when a mutation returns 401", async () => {
    mockSnapshot();
    mockedMoveTask.mockRejectedValueOnce(new ApiError("unauthorized", 401));

    const queryClient = renderPage({
      authSession: { authenticated: true, expiresAt: 1 },
    });

    fireEvent.click(screen.getByRole("button", { name: "Move to Active (Queue me)" }));

    await waitFor(() => expect(queryClient.getQueryData(authSessionQueryKey)).toBeNull());
  });

  it("exposes lane order semantics for assistive technology", () => {
    mockSnapshot({
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
          id: "d",
          title: "Queue next",
          note: "",
          due: "2026-04-23",
          priority: "high",
          status: "queued",
          sort_order: 1,
          lastUpdated: 4,
        },
        {
          id: "e",
          title: "Queue later",
          note: "",
          due: "2026-04-24",
          priority: "critical",
          status: "queued",
          sort_order: 2,
          lastUpdated: 5,
        },
      ],
    });

    renderPage();

    const queuedLane = screen.getByRole("region", { name: "Queued" });
    expect(within(queuedLane).getByText("Use Move up or Move down to reorder cards within the Queued lane.")).toBeInTheDocument();
    expect(
      within(queuedLane).getByText("Use Tab to reach one card shell, Arrow keys to move between cards, and Tab again to reach the action buttons."),
    ).toBeInTheDocument();

    const items = within(queuedLane).getAllByRole("listitem");
    expect(items).toHaveLength(3);
    expect(items[0]).toHaveAttribute("aria-posinset", "1");
    expect(items[0]).toHaveAttribute("aria-setsize", "3");
    expect(items[1]).toHaveAttribute("aria-posinset", "2");
    expect(items[2]).toHaveAttribute("aria-posinset", "3");
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
    ...overrides,
  };

  mockedUseBoardSnapshot.mockReturnValue({
    data,
    error: null,
    isPending: false,
  } as ReturnType<typeof useBoardSnapshot>);
}

function createDeferred<T>() {
  let reject!: (reason?: unknown) => void;
  let resolve!: (value: T | PromiseLike<T>) => void;

  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });

  return { promise, reject, resolve };
}
