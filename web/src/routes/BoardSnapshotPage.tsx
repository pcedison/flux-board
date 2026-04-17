import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { DndContext, MouseSensor, TouchSensor, type DragEndEvent, useSensor, useSensors } from "@dnd-kit/core";

import type { TaskPriority } from "../lib/api";
import { QueryState } from "../components/QueryState";
import { useBoardMutations } from "../lib/useBoardMutations";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";
import { BoardArchivePanel, BoardComposerPanel, BoardLane, BoardStatusBanner } from "../components/board";
import type { FocusTarget, MoveTaskRequest, BoardLaneDescriptor } from "../components/board";
import { getSameLaneDragMove } from "../components/board/dragAndDrop";

const lanes: BoardLaneDescriptor[] = [
  { label: "Queued", status: "queued" },
  { label: "Active", status: "active" },
  { label: "Done", status: "done" },
];

type BoardSnapshotData = NonNullable<ReturnType<typeof useBoardSnapshot>["data"]>;

export function BoardSnapshotPage() {
  const snapshot = useBoardSnapshot();
  const mutations = useBoardMutations();

  return (
    <QueryState
      error={snapshot.error}
      errorTitle="Failed to load board snapshot"
      isPending={snapshot.isPending}
      loadingMessage="Loading the current lane breakdown from the Go API."
    >
      {snapshot.data ? <BoardSnapshotContent data={snapshot.data} mutations={mutations} /> : null}
    </QueryState>
  );
}

function BoardSnapshotContent({
  data,
  mutations,
}: {
  data: BoardSnapshotData;
  mutations: ReturnType<typeof useBoardMutations>;
}) {
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 125, tolerance: 8 } }),
  );
  const titleInputRef = useRef<HTMLInputElement>(null);
  const dueInputRef = useRef<HTMLInputElement>(null);
  const cardRefs = useRef(new Map<string, HTMLElement>());
  const archiveButtonRefs = useRef(new Map<string, HTMLButtonElement>());
  const [title, setTitle] = useState("");
  const [due, setDue] = useState("");
  const [note, setNote] = useState("");
  const [priority, setPriority] = useState<TaskPriority>("medium");
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionStatus, setActionStatus] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<{ due?: string; title?: string }>({});
  const [focusTarget, setFocusTarget] = useState<FocusTarget | null>(null);
  const [activeCardId, setActiveCardId] = useState<string | null>(() => getFirstVisibleTaskId(data.tasks));

  const isBusy =
    mutations.createTask.isPending ||
    mutations.moveTask.isPending ||
    mutations.archiveTask.isPending ||
    mutations.restoreTask.isPending;
  const pendingMoveTaskID = mutations.moveTask.isPending ? mutations.moveTask.variables?.id : null;
  const pendingArchiveTaskID = mutations.archiveTask.isPending ? mutations.archiveTask.variables : null;
  const pendingRestoreTaskID = mutations.restoreTask.isPending ? mutations.restoreTask.variables : null;
  const tasksByLane = new Map(lanes.map((lane) => [lane.status, data.tasks.filter((task) => task.status === lane.status)]));

  useEffect(() => {
    setActiveCardId((current) => {
      if (current && data.tasks.some((task) => task.id === current)) {
        return current;
      }
      return getFirstVisibleTaskId(data.tasks);
    });
  }, [data.tasks]);

  useLayoutEffect(() => {
    if (!focusTarget) {
      return;
    }

    if (focusTarget.kind === "title") {
      titleInputRef.current?.focus();
      setFocusTarget(null);
      return;
    }

    if (focusTarget.kind === "task") {
      const taskSnapshot = data.tasks.find((task) => task.id === focusTarget.id);
      if (!taskSnapshot) {
        return;
      }

      if (focusTarget.status && taskSnapshot.status !== focusTarget.status) {
        return;
      }

      if (focusTarget.lastUpdated && taskSnapshot.lastUpdated < focusTarget.lastUpdated) {
        return;
      }

      if (activeCardId !== focusTarget.id) {
        setActiveCardId(focusTarget.id);
        return;
      }

      const taskCard = cardRefs.current.get(focusTarget.id);
      if (taskCard) {
        taskCard.focus();
        setFocusTarget(null);
      }
      return;
    }

    const archiveButton = archiveButtonRefs.current.get(focusTarget.id);
    if (archiveButton) {
      archiveButton.focus();
      setFocusTarget(null);
    }
  }, [activeCardId, data.archived, data.tasks, focusTarget]);

  function setCardRef(id: string, element: HTMLElement | null) {
    if (element) {
      cardRefs.current.set(id, element);
      return;
    }
    cardRefs.current.delete(id);
  }

  function setArchiveButtonRef(id: string, element: HTMLButtonElement | null) {
    if (element) {
      archiveButtonRefs.current.set(id, element);
      return;
    }
    archiveButtonRefs.current.delete(id);
  }

  function isTaskBusy(id: string) {
    return pendingMoveTaskID === id || pendingArchiveTaskID === id;
  }

  async function handleCreateTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedTitle = title.trim();

    if (!trimmedTitle || !due) {
      const nextFieldErrors: { due?: string; title?: string } = {};
      if (!trimmedTitle) {
        nextFieldErrors.title = "Enter a task title before creating a card.";
      }
      if (!due) {
        nextFieldErrors.due = "Choose a due date before creating a card.";
      }
      setFieldErrors(nextFieldErrors);
      setActionError(null);
      setActionStatus(null);
      if (nextFieldErrors.title) {
        titleInputRef.current?.focus();
      } else if (nextFieldErrors.due) {
        dueInputRef.current?.focus();
      }
      return;
    }

    setFieldErrors({});
    setActionError(null);
    setActionStatus(null);

    try {
      await mutations.createTask.mutateAsync({
        title: trimmedTitle,
        due,
        note: note.trim(),
        priority,
      });
      setTitle("");
      setDue("");
      setNote("");
      setPriority("medium");
      setActionStatus(`Created ${trimmedTitle} in the queued lane.`);
      setFocusTarget({ kind: "title" });
    } catch (error) {
      setActionError(readErrorMessage(error));
      setActionStatus(null);
    }
  }

  async function handleMoveTask(move: MoveTaskRequest, announcement: string) {
    setActionError(null);
    setActionStatus(null);
    try {
      const movedTask = await mutations.moveTask.mutateAsync(move);
      setActionStatus(announcement);
      setFocusTarget({
        kind: "task",
        id: movedTask.id,
        status: movedTask.status,
        lastUpdated: movedTask.lastUpdated,
      });
    } catch (error) {
      setActionError(readErrorMessage(error));
      setActionStatus(null);
    }
  }

  async function handleArchiveTask(id: string, taskTitle: string) {
    setActionError(null);
    setActionStatus(null);
    try {
      await mutations.archiveTask.mutateAsync(id);
      setActionStatus(`Archived ${taskTitle}.`);
      setFocusTarget({ kind: "archived", id });
    } catch (error) {
      setActionError(readErrorMessage(error));
      setActionStatus(null);
    }
  }

  async function handleRestoreTask(id: string, taskTitle: string, status: BoardSnapshotData["archived"][number]["status"]) {
    setActionError(null);
    setActionStatus(null);
    try {
      const restoredTask = await mutations.restoreTask.mutateAsync(id);
      setActionStatus(`Restored ${taskTitle} to ${status}.`);
      setFocusTarget({
        kind: "task",
        id: restoredTask.id,
        status: restoredTask.status,
        lastUpdated: restoredTask.lastUpdated,
      });
    } catch (error) {
      setActionError(readErrorMessage(error));
      setActionStatus(null);
    }
  }

  function handleCardNavigation(
    taskId: string,
    laneStatus: BoardSnapshotData["tasks"][number]["status"],
    index: number,
    key: "ArrowUp" | "ArrowDown" | "ArrowLeft" | "ArrowRight",
  ) {
    const currentLaneIndex = lanes.findIndex((lane) => lane.status === laneStatus);
    if (currentLaneIndex === -1) {
      return;
    }

    const currentLaneTasks = tasksByLane.get(laneStatus) ?? [];
    let targetTaskId: string | undefined;

    if (key === "ArrowUp") {
      if (index > 0) {
        targetTaskId = currentLaneTasks[index - 1]?.id;
      }
    } else if (key === "ArrowDown") {
      if (index < currentLaneTasks.length - 1) {
        targetTaskId = currentLaneTasks[index + 1]?.id;
      }
    } else {
      const laneStep = key === "ArrowLeft" ? -1 : 1;

      for (let nextLaneIndex = currentLaneIndex + laneStep; nextLaneIndex >= 0 && nextLaneIndex < lanes.length; nextLaneIndex += laneStep) {
        const nextLane = lanes[nextLaneIndex];
        const nextLaneTasks = tasksByLane.get(nextLane.status) ?? [];
        if (nextLaneTasks.length === 0) {
          continue;
        }

        const targetIndex = Math.min(index, nextLaneTasks.length - 1);
        targetTaskId = nextLaneTasks[targetIndex]?.id;
        break;
      }
    }

    if (!targetTaskId || targetTaskId === taskId) {
      return;
    }

    setFocusTarget({ kind: "task", id: targetTaskId });
  }

  function handleCardFocus(taskId: string) {
    setActiveCardId(taskId);
  }

  function handleDragEnd(event: DragEndEvent) {
    const activeId = String(event.active.id);
    const overId = event.over?.id ? String(event.over.id) : null;

    if (!overId || activeId === overId) {
      return;
    }

    for (const lane of lanes) {
      const laneTasks = data.tasks.filter((task) => task.status === lane.status);
      if (!laneTasks.some((task) => task.id === activeId) || !laneTasks.some((task) => task.id === overId)) {
        continue;
      }

      const move = getSameLaneDragMove(laneTasks, activeId, overId);
      if (!move) {
        return;
      }

      const movedTask = laneTasks.find((task) => task.id === activeId);
      if (!movedTask) {
        return;
      }

      void handleMoveTask(move, `Moved ${movedTask.title} within ${lane.label}.`);
      return;
    }
  }

  return (
    <DndContext sensors={sensors} onDragEnd={handleDragEnd}>
      <div className="board-grid" aria-busy={isBusy}>
        <BoardStatusBanner error={actionError} status={actionStatus} />

        {lanes.map((lane) => {
          const tasks = data.tasks.filter((task) => task.status === lane.status);

          return (
            <BoardLane
              key={lane.status}
              isTaskBusy={isTaskBusy}
              lane={lane}
              activeCardId={activeCardId}
              onCardFocus={handleCardFocus}
              onCardNavigate={handleCardNavigation}
              onArchiveTask={handleArchiveTask}
              onMoveTask={handleMoveTask}
              setCardRef={setCardRef}
              tasks={tasks}
            />
          );
        })}

        <section className="panel panel-secondary board-side-panel">
          <BoardComposerPanel
            due={due}
            dueInputRef={dueInputRef}
            fieldErrors={fieldErrors}
            isPending={mutations.createTask.isPending}
            note={note}
            onDueChange={(value) => {
              setDue(value);
              if (fieldErrors.due) {
                setFieldErrors((current) => ({ ...current, due: undefined }));
              }
            }}
            onNoteChange={setNote}
            onPriorityChange={setPriority}
            onSubmit={handleCreateTask}
            onTitleChange={(value) => {
              setTitle(value);
              if (fieldErrors.title) {
                setFieldErrors((current) => ({ ...current, title: undefined }));
              }
            }}
            priority={priority}
            title={title}
            titleInputRef={titleInputRef}
          />
          <BoardArchivePanel
            archived={data.archived}
            onRestoreTask={handleRestoreTask}
            pendingRestoreTaskID={pendingRestoreTaskID}
            setArchiveButtonRef={setArchiveButtonRef}
          />
        </section>
      </div>
    </DndContext>
  );
}

function readErrorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }
  return "The board action failed.";
}

function getFirstVisibleTaskId(tasks: BoardSnapshotData["tasks"]) {
  for (const lane of lanes) {
    const task = tasks.find((entry) => entry.status === lane.status);
    if (task) {
      return task.id;
    }
  }

  return null;
}
