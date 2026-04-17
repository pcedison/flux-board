import { describe, expect, it } from "vitest";

import type { Task } from "../../lib/api";
import { getSameLaneDragMove } from "./dragAndDrop";

describe("getSameLaneDragMove", () => {
  it("returns a move request for downward reorder", () => {
    const tasks = buildTasks();

    expect(getSameLaneDragMove(tasks, "a", "c")).toEqual({
      id: "a",
      status: "queued",
      anchorTaskId: "c",
      placeAfter: true,
    });
  });

  it("returns a move request for upward reorder", () => {
    const tasks = buildTasks();

    expect(getSameLaneDragMove(tasks, "c", "a")).toEqual({
      id: "c",
      status: "queued",
      anchorTaskId: "a",
      placeAfter: false,
    });
  });

  it("returns null for a self-drop", () => {
    const tasks = buildTasks();

    expect(getSameLaneDragMove(tasks, "b", "b")).toBeNull();
  });

  it("returns null when the over target is missing", () => {
    const tasks = buildTasks();

    expect(getSameLaneDragMove(tasks, "b", null)).toBeNull();
    expect(getSameLaneDragMove(tasks, "b", "missing")).toBeNull();
  });

  it("returns null when the active task is missing", () => {
    const tasks = buildTasks();

    expect(getSameLaneDragMove(tasks, "missing", "a")).toBeNull();
  });

  it("returns null for cross-lane moves", () => {
    const tasks = buildTasks();

    expect(getSameLaneDragMove(tasks, "a", "d")).toBeNull();
  });
});

function buildTasks(): Task[] {
  return [
    {
      id: "a",
      title: "Queue first",
      note: "",
      due: "2026-04-20",
      priority: "medium",
      status: "queued",
      sort_order: 0,
      lastUpdated: 1,
    },
    {
      id: "b",
      title: "Queue second",
      note: "",
      due: "2026-04-21",
      priority: "high",
      status: "queued",
      sort_order: 1,
      lastUpdated: 2,
    },
    {
      id: "c",
      title: "Queue third",
      note: "",
      due: "2026-04-22",
      priority: "critical",
      status: "queued",
      sort_order: 2,
      lastUpdated: 3,
    },
    {
      id: "d",
      title: "Active lane task",
      note: "",
      due: "2026-04-23",
      priority: "medium",
      status: "active",
      sort_order: 0,
      lastUpdated: 4,
    },
  ];
}
