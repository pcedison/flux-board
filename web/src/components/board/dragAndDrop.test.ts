import { describe, expect, it } from "vitest";

import type { Task } from "../../lib/api";
import { getDragMove, getLaneDropId } from "./dragAndDrop";

describe("getDragMove", () => {
  it("returns a move request for downward reorder", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "a", "c")).toEqual({
      id: "a",
      status: "queued",
      anchorTaskId: "c",
      placeAfter: true,
    });
  });

  it("returns a move request for upward reorder", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "c", "a")).toEqual({
      id: "c",
      status: "queued",
      anchorTaskId: "a",
      placeAfter: false,
    });
  });

  it("returns a cross-lane move when dropping on another task", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "a", "d")).toEqual({
      id: "a",
      status: "active",
      anchorTaskId: "d",
      placeAfter: false,
    });
  });

  it("returns a lane move when dropping on an empty lane", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "a", getLaneDropId("done"))).toEqual({
      id: "a",
      status: "done",
    });
  });

  it("returns a lane move to the end of a populated lane", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "a", getLaneDropId("active"))).toEqual({
      id: "a",
      status: "active",
      anchorTaskId: "d",
      placeAfter: true,
    });
  });

  it("returns null for a self-drop", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "b", "b")).toBeNull();
  });

  it("returns null when the over target is missing", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "b", null)).toBeNull();
    expect(getDragMove(tasks, "b", "missing")).toBeNull();
  });

  it("returns null when the active task is missing", () => {
    const tasks = buildTasks();

    expect(getDragMove(tasks, "missing", "a")).toBeNull();
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
