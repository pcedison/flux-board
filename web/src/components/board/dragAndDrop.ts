import type { Task } from "../../lib/api";
import type { MoveTaskRequest } from "./types";

export function getSameLaneDragMove(tasks: Task[], activeId: string, overId: string | null): MoveTaskRequest | null {
  if (!overId || activeId === overId) {
    return null;
  }

  const activeTask = tasks.find((task) => task.id === activeId);
  const overTask = tasks.find((task) => task.id === overId);

  if (!activeTask || !overTask || activeTask.status !== overTask.status) {
    return null;
  }

  const activeIndex = tasks.findIndex((task) => task.id === activeId);
  const overIndex = tasks.findIndex((task) => task.id === overId);

  if (activeIndex === -1 || overIndex === -1) {
    return null;
  }

  return {
    id: activeTask.id,
    status: activeTask.status,
    anchorTaskId: overTask.id,
    placeAfter: activeIndex < overIndex,
  };
}
