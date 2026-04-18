import { DndContext } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { Task } from "../../lib/api";
import { BoardTaskCard } from "./BoardTaskCard";

describe("BoardTaskCard", () => {
  it("renders queued task actions and dispatches lane reorder, lane move, and archive callbacks", () => {
    const onMoveTask = vi.fn();
    const onArchiveTask = vi.fn();
    const tasks = buildQueuedTasks();

    renderCard({
      index: 1,
      isActive: true,
      isBusy: false,
      laneLabel: "Queued",
      laneStatus: "queued",
      onCardFocus: () => {},
      onCardNavigate: () => {},
      onArchiveTask,
      onEditTask: vi.fn(),
      onMoveTask,
      setRef: () => {},
      task: tasks[1],
      tasks,
    });

    expect(screen.getByRole("button", { name: "Drag Queue next to reorder within Queued" })).toBeEnabled();
    expect(screen.getByText("Queue next").closest("article")).toHaveAttribute("tabindex", "0");
    fireEvent.click(screen.getByRole("button", { name: "Move Queue next up within Queued" }));
    expect(onMoveTask).toHaveBeenNthCalledWith(
      1,
      {
        id: "b",
        status: "queued",
        anchorTaskId: "a",
        placeAfter: false,
      },
      "Moved Queue next up within Queued.",
    );

    fireEvent.click(screen.getByRole("button", { name: "Move Queue next down within Queued" }));
    expect(onMoveTask).toHaveBeenNthCalledWith(
      2,
      {
        id: "b",
        status: "queued",
        anchorTaskId: "c",
        placeAfter: true,
      },
      "Moved Queue next down within Queued.",
    );

    fireEvent.click(screen.getByRole("button", { name: "Move to Active (Queue next)" }));
    expect(onMoveTask).toHaveBeenNthCalledWith(
      3,
      {
        id: "b",
        status: "active",
      },
      "Moved Queue next to Active.",
    );

    fireEvent.click(screen.getByRole("button", { name: "Archive Queue next" }));
    expect(onArchiveTask).toHaveBeenCalledWith("b", "Queue next");
  });

  it("disables all actions while the card is pending", () => {
    const tasks = buildQueuedTasks();

    renderCard({
      index: 1,
      isActive: true,
      isBusy: true,
      laneLabel: "Queued",
      laneStatus: "queued",
      onCardFocus: () => {},
      onCardNavigate: () => {},
      onArchiveTask: vi.fn(),
      onEditTask: vi.fn(),
      onMoveTask: vi.fn(),
      setRef: () => {},
      task: tasks[1],
      tasks,
    });

    expect(screen.getByRole("button", { name: "Drag Queue next to reorder within Queued" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Drag Queue next to reorder within Queued" })).toHaveAttribute("tabindex", "-1");
    expect(screen.getByText("Queue next").closest("article")).toHaveAttribute("tabindex", "0");
    expect(screen.getByRole("button", { name: "Move Queue next up within Queued" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Move Queue next down within Queued" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Move to Active (Queue next)" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Archive Queue next" })).toBeDisabled();
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
      note: "second",
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

function renderCard(props: Parameters<typeof BoardTaskCard>[0]) {
  render(
    <DndContext>
      <SortableContext items={props.tasks.map((task) => task.id)} strategy={verticalListSortingStrategy}>
        <BoardTaskCard {...props} />
      </SortableContext>
    </DndContext>,
  );
}
