import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { DndContext, MouseSensor, TouchSensor, closestCenter, type DragEndEvent, useSensor, useSensors } from "@dnd-kit/core";

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
  const searchInputRef = useRef<HTMLInputElement>(null);
  const cardRefs = useRef(new Map<string, HTMLElement>());
  const archiveButtonRefs = useRef(new Map<string, HTMLButtonElement>());
  const [title, setTitle] = useState("");
  const [due, setDue] = useState("");
  const [note, setNote] = useState("");
  const [priority, setPriority] = useState<TaskPriority>("medium");
  const [search, setSearch] = useState("");
  const [editTaskID, setEditTaskID] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editDue, setEditDue] = useState("");
  const [editNote, setEditNote] = useState("");
  const [editPriority, setEditPriority] = useState<TaskPriority>("medium");
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionStatus, setActionStatus] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<{ due?: string; title?: string }>({});
  const [editFieldErrors, setEditFieldErrors] = useState<{ due?: string; title?: string }>({});
  const [focusTarget, setFocusTarget] = useState<FocusTarget | null>(null);
  const filteredTasks = data.tasks.filter((task) => {
    const query = search.trim().toLowerCase();
    if (!query) {
      return true;
    }
    return `${task.title} ${task.note}`.toLowerCase().includes(query);
  });
  const [activeCardId, setActiveCardId] = useState<string | null>(() => getFirstVisibleTaskId(filteredTasks));

  const isBusy =
    mutations.createTask.isPending ||
    mutations.updateTask.isPending ||
    mutations.moveTask.isPending ||
    mutations.archiveTask.isPending ||
    mutations.restoreTask.isPending ||
    mutations.deleteArchivedTask.isPending;
  const pendingMoveTaskID = mutations.moveTask.isPending ? mutations.moveTask.variables?.id : null;
  const pendingUpdateTaskID = mutations.updateTask.isPending ? mutations.updateTask.variables?.id : null;
  const pendingArchiveTaskID = mutations.archiveTask.isPending ? mutations.archiveTask.variables : null;
  const pendingRestoreTaskID = mutations.restoreTask.isPending ? mutations.restoreTask.variables : null;
  const pendingDeleteArchivedTaskID = mutations.deleteArchivedTask.isPending ? mutations.deleteArchivedTask.variables : null;
  const tasksByLane = new Map(lanes.map((lane) => [lane.status, filteredTasks.filter((task) => task.status === lane.status)]));
  const visibleActiveCardId =
    activeCardId && filteredTasks.some((task) => task.id === activeCardId)
      ? activeCardId
      : getFirstVisibleTaskId(filteredTasks);
  const editingTaskAvailable = editTaskID ? data.tasks.some((task) => task.id === editTaskID) : false;

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const tagName = target?.tagName ?? "";
      const typingTarget = target?.isContentEditable || tagName === "INPUT" || tagName === "TEXTAREA" || tagName === "SELECT";
      if (typingTarget) {
        if (event.key === "Escape" && target === searchInputRef.current) {
          setSearch("");
          searchInputRef.current?.blur();
        }
        return;
      }

      if (event.key === "/") {
        event.preventDefault();
        searchInputRef.current?.focus();
      }
      if (event.key.toLowerCase() === "n") {
        event.preventDefault();
        titleInputRef.current?.focus();
      }
      if (event.key === "Escape" && search) {
        setSearch("");
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [search]);

  useLayoutEffect(() => {
    if (!focusTarget) {
      return;
    }

    const clearPendingFocus = () => {
      queueMicrotask(() => setFocusTarget(null));
    };

    if (focusTarget.kind === "title") {
      titleInputRef.current?.focus();
      clearPendingFocus();
      return;
    }

    if (focusTarget.kind === "task") {
      const taskSnapshot = filteredTasks.find((task) => task.id === focusTarget.id) ?? data.tasks.find((task) => task.id === focusTarget.id);
      if (!taskSnapshot) {
        return;
      }

      if (focusTarget.status && taskSnapshot.status !== focusTarget.status) {
        return;
      }

      if (focusTarget.lastUpdated && taskSnapshot.lastUpdated < focusTarget.lastUpdated) {
        return;
      }

      const taskCard = cardRefs.current.get(focusTarget.id);
      if (taskCard) {
        taskCard.focus();
        clearPendingFocus();
      }
      return;
    }

    const archiveButton = archiveButtonRefs.current.get(focusTarget.id);
    if (archiveButton) {
      archiveButton.focus();
      clearPendingFocus();
    }
  }, [data.archived, data.tasks, filteredTasks, focusTarget]);

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
    return pendingMoveTaskID === id || pendingArchiveTaskID === id || pendingUpdateTaskID === id;
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
    if (editTaskID === id) {
      setEditTaskID(null);
    }
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

  async function handleDeleteArchivedTask(id: string, taskTitle: string) {
    setActionError(null);
    setActionStatus(null);
    try {
      await mutations.deleteArchivedTask.mutateAsync(id);
      setActionStatus(`Deleted ${taskTitle} permanently.`);
    } catch (error) {
      setActionError(readErrorMessage(error));
      setActionStatus(null);
    }
  }

  async function handleUpdateTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editTaskID) {
      return;
    }

    const trimmedTitle = editTitle.trim();
    if (!trimmedTitle || !editDue) {
      const nextFieldErrors: { due?: string; title?: string } = {};
      if (!trimmedTitle) {
        nextFieldErrors.title = "Enter a task title before saving.";
      }
      if (!editDue) {
        nextFieldErrors.due = "Choose a due date before saving.";
      }
      setEditFieldErrors(nextFieldErrors);
      return;
    }

    setEditFieldErrors({});
    setActionError(null);
    setActionStatus(null);
    try {
      const updatedTask = await mutations.updateTask.mutateAsync({
        id: editTaskID,
        task: {
          title: trimmedTitle,
          due: editDue,
          note: editNote.trim(),
          priority: editPriority,
        },
      });
      setActionStatus(`Updated ${trimmedTitle}.`);
      setFocusTarget({
        kind: "task",
        id: updatedTask.id,
        status: updatedTask.status,
        lastUpdated: updatedTask.lastUpdated,
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
      const laneTasks = filteredTasks.filter((task) => task.status === lane.status);
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
    <DndContext collisionDetection={closestCenter} sensors={sensors} onDragEnd={handleDragEnd}>
      <div className="board-grid" aria-busy={isBusy}>
        <BoardStatusBanner error={actionError} status={actionStatus} />

        {lanes.map((lane) => {
          const tasks = filteredTasks.filter((task) => task.status === lane.status);

          return (
            <BoardLane
              key={lane.status}
              isTaskBusy={isTaskBusy}
              lane={lane}
              activeCardId={visibleActiveCardId}
              onCardFocus={handleCardFocus}
              onCardNavigate={handleCardNavigation}
              onArchiveTask={handleArchiveTask}
              onEditTask={(task) => {
                setEditTaskID(task.id);
                setEditTitle(task.title);
                setEditDue(task.due);
                setEditNote(task.note);
                setEditPriority(task.priority);
              }}
              onMoveTask={handleMoveTask}
              setCardRef={setCardRef}
              tasks={tasks}
            />
          );
        })}

        <section className="panel panel-secondary board-side-panel">
          <div className="panel panel-secondary board-search-panel">
            <h2>Search & shortcuts</h2>
            <p className="meta">Press <kbd>/</kbd> to search and <kbd>N</kbd> to jump back to the create form.</p>
            <label className="form-field" htmlFor="board-search">
              Search tasks
            </label>
            <input
              id="board-search"
              className="text-input"
              ref={searchInputRef}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Filter by title or note"
            />
          </div>
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
          {editTaskID && editingTaskAvailable ? (
            <section className="panel panel-secondary">
              <h2>Edit task</h2>
              <p className="meta">Adjust the selected card without changing its lane or ordering.</p>
              <form className="board-form" onSubmit={handleUpdateTask} noValidate>
                <label className="form-field" htmlFor="board-task-edit-title">
                  Title
                </label>
                <input
                  id="board-task-edit-title"
                  className="text-input"
                  value={editTitle}
                  onChange={(event) => {
                    setEditTitle(event.target.value);
                    if (editFieldErrors.title) {
                      setEditFieldErrors((current) => ({ ...current, title: undefined }));
                    }
                  }}
                  aria-invalid={Boolean(editFieldErrors.title)}
                  aria-describedby={editFieldErrors.title ? "board-task-edit-title-error" : undefined}
                />
                {editFieldErrors.title ? (
                  <p id="board-task-edit-title-error" className="form-error" role="alert">
                    {editFieldErrors.title}
                  </p>
                ) : null}

                <div className="field-grid">
                  <div>
                    <label className="form-field" htmlFor="board-task-edit-due">
                      Due date
                    </label>
                    <input
                      id="board-task-edit-due"
                      className="text-input"
                      type="date"
                      value={editDue}
                      onChange={(event) => {
                        setEditDue(event.target.value);
                        if (editFieldErrors.due) {
                          setEditFieldErrors((current) => ({ ...current, due: undefined }));
                        }
                      }}
                      aria-invalid={Boolean(editFieldErrors.due)}
                      aria-describedby={editFieldErrors.due ? "board-task-edit-due-error" : undefined}
                    />
                    {editFieldErrors.due ? (
                      <p id="board-task-edit-due-error" className="form-error" role="alert">
                        {editFieldErrors.due}
                      </p>
                    ) : null}
                  </div>
                  <div>
                    <label className="form-field" htmlFor="board-task-edit-priority">
                      Priority
                    </label>
                    <select
                      id="board-task-edit-priority"
                      className="text-input"
                      value={editPriority}
                      onChange={(event) => setEditPriority(event.target.value as TaskPriority)}
                    >
                      <option value="medium">medium</option>
                      <option value="high">high</option>
                      <option value="critical">critical</option>
                    </select>
                  </div>
                </div>

                <label className="form-field" htmlFor="board-task-edit-note">
                  Note
                </label>
                <textarea
                  id="board-task-edit-note"
                  className="text-input text-area"
                  rows={4}
                  value={editNote}
                  onChange={(event) => setEditNote(event.target.value)}
                />
                <div className="action-row">
                  <button className="nav-pill nav-pill-active auth-submit" type="submit" disabled={mutations.updateTask.isPending}>
                    {mutations.updateTask.isPending ? "Saving..." : "Save changes"}
                  </button>
                  <button
                    className="nav-pill nav-pill-muted nav-button"
                    type="button"
                    onClick={() => setEditTaskID(null)}
                  >
                    Close editor
                  </button>
                </div>
              </form>
            </section>
          ) : null}
          <BoardArchivePanel
            archived={data.archived}
            onDeleteArchivedTask={handleDeleteArchivedTask}
            onRestoreTask={handleRestoreTask}
            pendingDeleteArchivedTaskID={pendingDeleteArchivedTaskID}
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
