import type { Task } from "../../lib/api";
import type { TaskStatus } from "../../lib/api";
import type { MoveTaskRequest } from "./types";

const laneDropPrefix = "lane:";
const laneOrder: TaskStatus[] = ["queued", "active", "done"];

export function getLaneDropId(status: TaskStatus) {
  return `${laneDropPrefix}${status}`;
}

export function readLaneStatusFromDropId(id: string | null): TaskStatus | null {
  if (!id || !id.startsWith(laneDropPrefix)) {
    return null;
  }

  const status = id.slice(laneDropPrefix.length);
  if (status === "queued" || status === "active" || status === "done") {
    return status;
  }

  return null;
}

export function getDragMove(tasks: Task[], activeId: string, overId: string | null): MoveTaskRequest | null {
  if (!overId || activeId === overId) {
    return null;
  }

  const activeTask = tasks.find((task) => task.id === activeId);
  if (!activeTask) {
    return null;
  }

  const laneStatus = readLaneStatusFromDropId(overId);
  if (laneStatus) {
    return getLaneDropMove(tasks, activeTask, laneStatus);
  }

  const overTask = tasks.find((task) => task.id === overId);
  if (!overTask) {
    return null;
  }

  if (activeTask.status !== overTask.status) {
    return {
      id: activeTask.id,
      status: overTask.status,
      anchorTaskId: overTask.id,
      placeAfter: false,
    };
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

export function applyMoveToTasks(tasks: Task[], move: MoveTaskRequest): Task[] {
  const activeTask = tasks.find((task) => task.id === move.id);
  if (!activeTask) {
    return tasks;
  }

  const tasksByLane = {
    active: [] as Task[],
    done: [] as Task[],
    queued: [] as Task[],
  };

  for (const task of tasks) {
    if (task.id === move.id) {
      continue;
    }
    tasksByLane[task.status].push(task);
  }

  const nextTask: Task = {
    ...activeTask,
    status: move.status,
  };
  const targetLane = tasksByLane[move.status];
  let insertIndex = targetLane.length;

  if (move.anchorTaskId) {
    const anchorIndex = targetLane.findIndex((task) => task.id === move.anchorTaskId);
    if (anchorIndex !== -1) {
      insertIndex = move.placeAfter ? anchorIndex + 1 : anchorIndex;
    }
  }

  targetLane.splice(insertIndex, 0, nextTask);

  return laneOrder.flatMap((status) =>
    tasksByLane[status].map((task, index) => ({
      ...task,
      sort_order: index,
    })),
  );
}

function getLaneDropMove(tasks: Task[], activeTask: Task, targetStatus: TaskStatus): MoveTaskRequest | null {
  const targetTasks = tasks.filter((task) => task.status === targetStatus && task.id !== activeTask.id);
  const lastTask = targetTasks[targetTasks.length - 1];

  if (activeTask.status === targetStatus) {
    const sameLaneTasks = tasks.filter((task) => task.status === targetStatus);
    if (sameLaneTasks.length <= 1) {
      return null;
    }

    const isAlreadyLast = sameLaneTasks[sameLaneTasks.length - 1]?.id === activeTask.id;
    if (isAlreadyLast) {
      return null;
    }
  }

  if (!lastTask) {
    return {
      id: activeTask.id,
      status: targetStatus,
    };
  }

  return {
    id: activeTask.id,
    status: targetStatus,
    anchorTaskId: lastTask.id,
    placeAfter: true,
  };
}
