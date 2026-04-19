import { DndContext } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { Task } from "../../lib/api";
import { PreferencesProvider } from "../../lib/preferences";
import { BoardTaskCard } from "./BoardTaskCard";

describe("BoardTaskCard", () => {
  it("renders compact task content and lets the user select the card", () => {
    const onSelectTask = vi.fn();
    const task = buildQueuedTasks()[1];

    renderCard({
      index: 1,
      isActive: true,
      isBusy: false,
      isSelected: true,
      laneStatus: "queued",
      onCardFocus: () => {},
      onCardNavigate: () => {},
      onSelectTask,
      setRef: () => {},
      task,
    });

    const article = screen.getByText("Queue next").closest("article");
    expect(article).toHaveAttribute("tabindex", "0");
    expect(article).toHaveClass("card-selected");
    expect(screen.getByText("High")).toBeInTheDocument();
    expect(screen.getByText("Due 2026-04-21")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Drag Queue next/i })).not.toBeInTheDocument();

    fireEvent.click(article!);
    expect(onSelectTask).toHaveBeenCalledWith(task);
  });

  it("keeps the full card available while pending without rendering a separate drag button", () => {
    renderCard({
      index: 1,
      isActive: true,
      isBusy: true,
      isSelected: false,
      laneStatus: "queued",
      onCardFocus: () => {},
      onCardNavigate: () => {},
      onSelectTask: vi.fn(),
      setRef: () => {},
      task: buildQueuedTasks()[1],
    });

    expect(screen.getByText("Queue next").closest("article")).toHaveAttribute("tabindex", "0");
    expect(screen.queryByRole("button", { name: /Drag Queue next/i })).not.toBeInTheDocument();
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
    <PreferencesProvider>
      <DndContext>
        <SortableContext items={[props.task.id]} strategy={verticalListSortingStrategy}>
          <BoardTaskCard {...props} />
        </SortableContext>
      </DndContext>
    </PreferencesProvider>,
  );
}
