import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { Task } from "../../lib/api";
import { BoardLane } from "./BoardLane";

describe("BoardLane", () => {
  it("renders an empty state when the lane has no tasks", () => {
    render(
      <BoardLane
        isTaskBusy={() => false}
        lane={{ label: "Queued", status: "queued" }}
        onArchiveTask={vi.fn()}
        onEditTask={vi.fn()}
        onMoveTask={vi.fn()}
        setCardRef={vi.fn()}
        tasks={[]}
      />,
    );

    const lane = screen.getByRole("region", { name: "Queued" });
    expect(within(lane).getByText("No tasks in this lane yet.")).toBeInTheDocument();
    expect(within(lane).queryByRole("list")).not.toBeInTheDocument();
  });

  it("renders lane semantics and card content for non-empty lanes", () => {
    render(
      <BoardLane
        isTaskBusy={(id) => id === "b"}
        lane={{ label: "Queued", status: "queued" }}
        onArchiveTask={vi.fn()}
        onEditTask={vi.fn()}
        onMoveTask={vi.fn()}
        setCardRef={vi.fn()}
        tasks={buildQueuedTasks()}
      />,
    );

    const lane = screen.getByRole("region", { name: "Queued" });
    expect(within(lane).getByText("Use Move up or Move down to reorder cards within the Queued lane.")).toBeInTheDocument();
    expect(within(lane).getByText("Queue me")).toBeInTheDocument();
    expect(within(lane).getByText("Queue next")).toBeInTheDocument();
    expect(within(lane).getByText("Queue later")).toBeInTheDocument();

    const items = within(lane).getAllByRole("listitem");
    expect(items).toHaveLength(3);
    expect(items[0]).toHaveAttribute("aria-posinset", "1");
    expect(items[1]).toHaveAttribute("aria-posinset", "2");
    expect(items[2]).toHaveAttribute("aria-posinset", "3");
    expect(items[0]).toHaveAttribute("aria-setsize", "3");
    expect(items[1]).toHaveAttribute("aria-setsize", "3");
    expect(items[2]).toHaveAttribute("aria-setsize", "3");

    expect(screen.getByRole("button", { name: "Move Queue next up within Queued" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Archive Queue me" })).toBeEnabled();
  });
});

function buildQueuedTasks(): Task[] {
  return [
    {
      id: "a",
      title: "Queue me",
      note: "",
      due: "2026-04-20",
      priority: "medium",
      status: "queued",
      sort_order: 0,
      lastUpdated: 1,
    },
    {
      id: "b",
      title: "Queue next",
      note: "",
      due: "2026-04-21",
      priority: "high",
      status: "queued",
      sort_order: 1,
      lastUpdated: 2,
    },
    {
      id: "c",
      title: "Queue later",
      note: "",
      due: "2026-04-22",
      priority: "critical",
      status: "queued",
      sort_order: 2,
      lastUpdated: 3,
    },
  ];
}
