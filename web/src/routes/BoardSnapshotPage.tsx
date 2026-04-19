import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { DndContext, DragOverlay, PointerSensor, closestCenter, type DragEndEvent, type DragStartEvent, useSensor, useSensors } from "@dnd-kit/core";

import { QueryState } from "../components/QueryState";
import { BoardArchivePanel, BoardComposerPanel, BoardLane, BoardStatusBanner, BoardTaskCardPreview } from "../components/board";
import type { BoardLaneDescriptor, FocusTarget, MoveTaskRequest } from "../components/board";
import { applyMoveToTasks, getDragMove } from "../components/board/dragAndDrop";
import type { Task, TaskPriority } from "../lib/api";
import { usePreferences } from "../lib/preferences";
import { useBoardMutations } from "../lib/useBoardMutations";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

const laneStatuses: BoardLaneDescriptor["status"][] = ["queued", "active", "done"];

type BoardSnapshotData = NonNullable<ReturnType<typeof useBoardSnapshot>["data"]>;

export function BoardSnapshotPage() {
  const snapshot = useBoardSnapshot();
  const mutations = useBoardMutations();
  const { copy } = usePreferences();

  return (
    <QueryState
      error={snapshot.error}
      errorTitle={copy.board.errorTitle}
      isPending={snapshot.isPending}
      loadingMessage={copy.board.loadingMessage}
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
  const { copy, priorityLabel, statusLabel } = usePreferences();
  const lanes = laneStatuses.map((status) => ({ label: statusLabel(status), status }));
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
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
  const [activeTab, setActiveTab] = useState<'search' | 'new' | 'edit' | 'archive'>('new');
  const [boardTasks, setBoardTasks] = useState(data.tasks);
  const [activeDragTaskId, setActiveDragTaskId] = useState<string | null>(null);
  const filteredTasks = boardTasks.filter((task) => {
    const query = search.trim().toLowerCase();
    if (!query) {
      return true;
    }
    return `${task.title} ${task.note}`.toLowerCase().includes(query);
  });
  const [activeCardId, setActiveCardId] = useState<string | null>(() => getFirstVisibleTaskId(filteredTasks, lanes));

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
      : getFirstVisibleTaskId(filteredTasks, lanes);
  const selectedTask = editTaskID ? boardTasks.find((task) => task.id === editTaskID) ?? null : null;
  const activeDragTask = activeDragTaskId ? boardTasks.find((task) => task.id === activeDragTaskId) ?? null : null;

  useEffect(() => {
    setBoardTasks(data.tasks);
  }, [data.tasks]);

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
        setActiveTab('search');
        searchInputRef.current?.focus();
      }
      if (event.key.toLowerCase() === "n") {
        event.preventDefault();
        setActiveTab('new');
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
      const taskSnapshot = filteredTasks.find((task) => task.id === focusTarget.id) ?? boardTasks.find((task) => task.id === focusTarget.id);
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
  }, [boardTasks, data.archived, filteredTasks, focusTarget]);

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

  function selectTask(task: Task) {
    setEditTaskID(task.id);
    setEditTitle(task.title);
    setEditDue(task.due);
    setEditNote(task.note);
    setEditPriority(task.priority);
    setEditFieldErrors({});
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
        nextFieldErrors.title = copy.board.createTitleError;
      }
      if (!due) {
        nextFieldErrors.due = copy.board.createDueError;
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
      setActionStatus(copy.board.createdStatus(trimmedTitle, statusLabel("queued")));
      setFocusTarget({ kind: "title" });
    } catch (error) {
      setActionError(readErrorMessage(error, copy.board.defaultActionError));
      setActionStatus(null);
    }
  }

  async function handleMoveTask(move: MoveTaskRequest, announcement: string, previousTasks: Task[]) {
    setActionError(null);
    setActionStatus(null);
    try {
      const movedTask = await mutations.moveTask.mutateAsync(move);
      setBoardTasks((current) => current.map((task) => (task.id === movedTask.id ? { ...task, ...movedTask } : task)));
      setActionStatus(announcement);
      setFocusTarget({
        kind: "task",
        id: movedTask.id,
        status: movedTask.status,
        lastUpdated: movedTask.lastUpdated,
      });
    } catch (error) {
      setBoardTasks(previousTasks);
      setActionError(readErrorMessage(error, copy.board.defaultActionError));
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
      setActionStatus(copy.board.archivedStatus(taskTitle));
      setFocusTarget({ kind: "archived", id });
    } catch (error) {
      setActionError(readErrorMessage(error, copy.board.defaultActionError));
      setActionStatus(null);
    }
  }

  async function handleRestoreTask(id: string, taskTitle: string, status: BoardSnapshotData["archived"][number]["status"]) {
    setActionError(null);
    setActionStatus(null);
    try {
      const restoredTask = await mutations.restoreTask.mutateAsync(id);
      setActionStatus(copy.board.restoredStatus(taskTitle, statusLabel(status)));
      setFocusTarget({
        kind: "task",
        id: restoredTask.id,
        status: restoredTask.status,
        lastUpdated: restoredTask.lastUpdated,
      });
    } catch (error) {
      setActionError(readErrorMessage(error, copy.board.defaultActionError));
      setActionStatus(null);
    }
  }

  async function handleDeleteArchivedTask(id: string, taskTitle: string) {
    setActionError(null);
    setActionStatus(null);
    try {
      await mutations.deleteArchivedTask.mutateAsync(id);
      setActionStatus(copy.board.deletedStatus(taskTitle));
    } catch (error) {
      setActionError(readErrorMessage(error, copy.board.defaultActionError));
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
        nextFieldErrors.title = copy.board.updateTitleError;
      }
      if (!editDue) {
        nextFieldErrors.due = copy.board.updateDueError;
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
      setActionStatus(copy.board.updatedStatus(trimmedTitle));
      selectTask(updatedTask);
      setFocusTarget({
        kind: "task",
        id: updatedTask.id,
        status: updatedTask.status,
        lastUpdated: updatedTask.lastUpdated,
      });
    } catch (error) {
      setActionError(readErrorMessage(error, copy.board.defaultActionError));
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
    const task = boardTasks.find((entry) => entry.id === taskId);
    if (task) {
      selectTask(task);
      setActiveTab('edit');
    }
  }

  function handleDragStart(event: DragStartEvent) {
    setActiveDragTaskId(String(event.active.id));
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveDragTaskId(null);
    const activeId = String(event.active.id);
    const overId = event.over?.id ? String(event.over.id) : null;
    const move = getDragMove(boardTasks, activeId, overId);
    if (!move) {
      return;
    }

    const movedTask = boardTasks.find((task) => task.id === activeId);
    if (!movedTask) {
      return;
    }

    const announcement =
      move.status === movedTask.status
        ? copy.board.movedWithinStatus(movedTask.title, statusLabel(move.status))
        : copy.board.movedToStatus(movedTask.title, statusLabel(move.status));

    const previousTasks = boardTasks;
    setBoardTasks((current) => applyMoveToTasks(current, move));
    void handleMoveTask(move, announcement, previousTasks);
  }

  return (
    <DndContext
      collisionDetection={closestCenter}
      sensors={sensors}
      onDragEnd={handleDragEnd}
      onDragStart={handleDragStart}
      onDragCancel={() => setActiveDragTaskId(null)}
    >
      <div className="board-grid" aria-busy={isBusy}>
        <BoardStatusBanner error={actionError} status={actionStatus} />

        {lanes.map((lane) => {
          const tasks = filteredTasks.filter((task) => task.status === lane.status);

          return (
            <BoardLane
              key={lane.status}
              activeCardId={visibleActiveCardId}
              isTaskBusy={isTaskBusy}
              lane={lane}
              onCardFocus={handleCardFocus}
              onCardNavigate={handleCardNavigation}
              onSelectTask={(task) => {
                selectTask(task);
                setActiveTab('edit');
              }}
              selectedTaskId={editTaskID}
              setCardRef={setCardRef}
              tasks={tasks}
            />
          );
        })}

        <aside className="board-side-panel">
          <div className="panel-tabs">
            {(['search', 'new', 'edit', 'archive'] as const).map((t) => (
              <button
                key={t}
                className={`panel-tab${activeTab === t ? ' panel-tab-active' : ''}`}
                onClick={() => setActiveTab(t)}
              >
                {{ search: '搜尋', new: '新增', edit: '編輯', archive: '封存' }[t]}
              </button>
            ))}
          </div>

          <div className="panel-body">
            {activeTab === 'search' && (
              <div className="board-form">
                <h2>{copy.board.searchTitle}</h2>
                <p className="meta">{copy.board.searchHint}</p>
                <label className="form-field" htmlFor="board-search">
                  {copy.board.searchLabel}
                </label>
                <input
                  id="board-search"
                  className="text-input"
                  ref={searchInputRef}
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder={copy.board.searchPlaceholder}
                />
              </div>
            )}
            {activeTab === 'new' && (
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
            )}
            {activeTab === 'edit' && (
              selectedTask ? (
                <section>
                  <h2>{copy.board.selectedTaskTitle}</h2>
                  <p className="meta">{copy.board.selectedTaskHint}</p>
                  <form className="board-form" onSubmit={handleUpdateTask} noValidate>
                    <label className="form-field" htmlFor="board-task-edit-title">
                      {copy.common.title}
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
                          {copy.common.dueDate}
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
                          {copy.common.priority}
                        </label>
                        <select
                          id="board-task-edit-priority"
                          className="text-input"
                          value={editPriority}
                          onChange={(event) => setEditPriority(event.target.value as TaskPriority)}
                        >
                          <option value="medium">{priorityLabel("medium")}</option>
                          <option value="high">{priorityLabel("high")}</option>
                          <option value="critical">{priorityLabel("critical")}</option>
                        </select>
                      </div>
                    </div>

                    <label className="form-field" htmlFor="board-task-edit-note">
                      {copy.common.note}
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
                        {mutations.updateTask.isPending ? copy.board.saveChangesPending : copy.board.saveChanges}
                      </button>
                      <button
                        className="nav-pill nav-pill-muted nav-button"
                        type="button"
                        onClick={() => setEditTaskID(null)}
                      >
                        {copy.board.clearSelection}
                      </button>
                      <button
                        className="nav-pill nav-pill-muted nav-button"
                        type="button"
                        disabled={mutations.archiveTask.isPending}
                        onClick={() => {
                          void handleArchiveTask(selectedTask.id, selectedTask.title);
                        }}
                      >
                        {mutations.archiveTask.isPending ? copy.board.archiveTaskPending : copy.board.archiveTask}
                      </button>
                    </div>
                  </form>
                </section>
              ) : (
                <div className="panel-placeholder">
                  <p className="meta">{copy.board.selectedTaskEmpty}</p>
                </div>
              )
            )}
            {activeTab === 'archive' && (
              <BoardArchivePanel
                archived={data.archived}
                onDeleteArchivedTask={handleDeleteArchivedTask}
                onRestoreTask={handleRestoreTask}
                pendingDeleteArchivedTaskID={pendingDeleteArchivedTaskID}
                pendingRestoreTaskID={pendingRestoreTaskID}
                setArchiveButtonRef={setArchiveButtonRef}
              />
            )}
          </div>
        </aside>
      </div>
      <DragOverlay
        dropAnimation={{
          duration: 180,
          easing: "cubic-bezier(0.2, 0, 0, 1)",
        }}
      >
        {activeDragTask ? <BoardTaskCardPreview task={activeDragTask} /> : null}
      </DragOverlay>
    </DndContext>
  );
}

function readErrorMessage(error: unknown, fallback: string) {
  if (error instanceof Error) {
    return error.message;
  }
  return fallback;
}

function getFirstVisibleTaskId(tasks: BoardSnapshotData["tasks"], lanes: BoardLaneDescriptor[]) {
  for (const lane of lanes) {
    const task = tasks.find((entry) => entry.status === lane.status);
    if (task) {
      return task.id;
    }
  }

  return null;
}
