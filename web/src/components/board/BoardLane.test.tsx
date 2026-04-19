import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { Task } from "../../lib/api";
import { BoardLane } from "./BoardLane";

describe("BoardLane", () => {
  it("renders an empty state when the lane has no tasks", () => {
    render(
      <BoardLane
        isTaskBusy={() => false}
        lane={{ label: "Queued", status: "queued" }}
        onSelectTask={vi.fn()}
        setCardRef={vi.fn()}
        tasks={[]}
      />,
    );

    const lane = screen.getByRole("region", { name: "Queued" });
    expect(within(lane).getByText("No tasks in this lane yet.")).toBeInTheDocument();
    expect(within(lane).queryByRole("list")).not.toBeInTheDocument();
  });

  it("collapses multi-task lanes until the user expands them", () => {
    render(
      <BoardLane
        isTaskBusy={(id) => id === "b"}
        lane={{ label: "Queued", status: "queued" }}
        onSelectTask={vi.fn()}
        selectedTaskId="a"
        setCardRef={vi.fn()}
        tasks={buildQueuedTasks()}
      />,
    );

    const lane = screen.getByRole("region", { name: "Queued" });
    expect(within(lane).getByRole("button", { name: /3 tasks hidden/i })).toBeInTheDocument();
    expect(within(lane).queryByRole("list")).not.toBeInTheDocument();

    fireEvent.click(within(lane).getByRole("button", { name: /3 tasks hidden/i }));

    expect(within(lane).getByText("Queue me")).toBeInTheDocument();
    expect(within(lane).getByText("Queue next")).toBeInTheDocument();
    expect(within(lane).getByText("Queue later")).toBeInTheDocument();

    const items = within(lane).getAllByRole("listitem");
    expect(items).toHaveLength(3);
    expect(items[0]).toHaveAttribute("aria-posinset", "1");
    expect(items[1]).toHaveAttribute("aria-posinset", "2");
    expect(items[2]).toHaveAttribute("aria-posinset", "3");

    expect(screen.getByRole("button", { name: "Drag Queue next to reorder or move it from Queued" })).toBeDisabled();
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
