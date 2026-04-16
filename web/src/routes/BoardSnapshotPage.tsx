import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import type { TaskPriority, TaskStatus } from "../lib/api";
import { QueryState } from "../components/QueryState";
import { useBoardMutations } from "../lib/useBoardMutations";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

const lanes: Array<{ label: string; status: TaskStatus }> = [
  { label: "Queued", status: "queued" },
  { label: "Active", status: "active" },
  { label: "Done", status: "done" },
];

const priorityOptions: TaskPriority[] = ["medium", "high", "critical"];

const moveTargets: Record<TaskStatus, Array<{ label: string; status: TaskStatus }>> = {
  queued: [{ label: "Move to Active", status: "active" }],
  active: [
    { label: "Back to Queued", status: "queued" },
    { label: "Move to Done", status: "done" },
  ],
  done: [{ label: "Back to Active", status: "active" }],
};

type MoveTaskRequest = {
  anchorTaskId?: string;
  id: string;
  placeAfter?: boolean;
  status: TaskStatus;
};

type FocusTarget =
  | { kind: "archived"; id: string }
  | { kind: "task"; id: string }
  | { kind: "title" };

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
  data: NonNullable<ReturnType<typeof useBoardSnapshot>["data"]>;
  mutations: ReturnType<typeof useBoardMutations>;
}) {
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

  const isBusy =
    mutations.createTask.isPending ||
    mutations.moveTask.isPending ||
    mutations.archiveTask.isPending ||
    mutations.restoreTask.isPending;
  const pendingMoveTaskID = mutations.moveTask.isPending ? mutations.moveTask.variables?.id : null;
  const pendingArchiveTaskID = mutations.archiveTask.isPending ? mutations.archiveTask.variables : null;
  const pendingRestoreTaskID = mutations.restoreTask.isPending ? mutations.restoreTask.variables : null;

  useEffect(() => {
    if (!focusTarget) {
      return;
    }

    if (focusTarget.kind === "title") {
      titleInputRef.current?.focus();
      setFocusTarget(null);
      return;
    }

    if (focusTarget.kind === "task") {
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
  }, [data.archived, data.tasks, focusTarget]);

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
      await mutations.moveTask.mutateAsync(move);
      setActionStatus(announcement);
      setFocusTarget({ kind: "task", id: move.id });
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

  async function handleRestoreTask(id: string, taskTitle: string, status: TaskStatus) {
    setActionError(null);
    setActionStatus(null);
    try {
      await mutations.restoreTask.mutateAsync(id);
      setActionStatus(`Restored ${taskTitle} to ${status}.`);
      setFocusTarget({ kind: "task", id });
    } catch (error) {
      setActionError(readErrorMessage(error));
      setActionStatus(null);
    }
  }

  return (
    <div className="board-grid" aria-busy={isBusy}>
      {actionError ? (
        <section className="panel panel-error board-feedback" role="alert">
          <strong>Board action failed.</strong>
          <p>{actionError}</p>
        </section>
      ) : null}

      {!actionError && actionStatus ? (
        <section className="panel panel-status board-feedback" role="status">
          <strong>Board updated.</strong>
          <p>{actionStatus}</p>
        </section>
      ) : null}

      {lanes.map((lane) => {
        const tasks = data.tasks.filter((task) => task.status === lane.status);

        return (
          <section key={lane.status} className="lane" aria-labelledby={`lane-${lane.status}`}>
            <div className="lane-head">
              <h2 id={`lane-${lane.status}`}>{lane.label}</h2>
              <span>{tasks.length}</span>
            </div>

            {tasks.length === 0 ? (
              <p className="empty">No tasks in this lane yet.</p>
            ) : (
              <>
                <p id={`lane-${lane.status}-hint`} className="visually-hidden">
                  Use Move up or Move down to reorder cards within the {lane.label} lane.
                </p>
                <ol className="lane-list" aria-describedby={`lane-${lane.status}-hint`}>
                {tasks.map((task, index) => (
                  <li
                    key={task.id}
                    className="lane-list-item"
                    aria-posinset={index + 1}
                    aria-setsize={tasks.length}
                  >
                    <article
                      className={`card${isTaskBusy(task.id) ? " card-pending" : ""}`}
                      ref={(element) => setCardRef(task.id, element)}
                      tabIndex={-1}
                    >
                      <div className="card-row">
                        <strong>{task.title}</strong>
                        <span className={`priority priority-${task.priority}`}>{task.priority}</span>
                      </div>
                      <p className="meta">Due {task.due} / order {task.sort_order}</p>
                      {task.note ? <p className="card-note">{task.note}</p> : null}
                      <div className="card-actions">
                        {index > 0 ? (
                          <button
                            className="action-button"
                            type="button"
                            disabled={isTaskBusy(task.id)}
                            aria-label={`Move ${task.title} up within ${lane.label}`}
                            onClick={() => {
                              const previousTask = tasks[index - 1];
                              void handleMoveTask(
                                {
                                  id: task.id,
                                  status: task.status,
                                  anchorTaskId: previousTask.id,
                                  placeAfter: false,
                                },
                                `Moved ${task.title} up within ${lane.label}.`,
                              );
                            }}
                          >
                            Move up
                          </button>
                        ) : null}
                        {index < tasks.length - 1 ? (
                          <button
                            className="action-button"
                            type="button"
                            disabled={isTaskBusy(task.id)}
                            aria-label={`Move ${task.title} down within ${lane.label}`}
                            onClick={() => {
                              const nextTask = tasks[index + 1];
                              void handleMoveTask(
                                {
                                  id: task.id,
                                  status: task.status,
                                  anchorTaskId: nextTask.id,
                                  placeAfter: true,
                                },
                                `Moved ${task.title} down within ${lane.label}.`,
                              );
                            }}
                          >
                            Move down
                          </button>
                        ) : null}
                        {moveTargets[task.status].map((target) => (
                          <button
                            key={target.status}
                            className="action-button"
                            type="button"
                            disabled={isTaskBusy(task.id)}
                            aria-label={`${target.label} (${task.title})`}
                            onClick={() => {
                              const targetLabel = target.label
                                .replace("Move to ", "")
                                .replace("Back to ", "");
                              void handleMoveTask(
                                {
                                  id: task.id,
                                  status: target.status,
                                },
                                `Moved ${task.title} to ${targetLabel}.`,
                              );
                            }}
                          >
                            {target.label}
                          </button>
                        ))}
                        <button
                          className="action-button action-button-secondary"
                          type="button"
                          disabled={isTaskBusy(task.id)}
                          aria-label={`Archive ${task.title}`}
                          onClick={() => {
                            void handleArchiveTask(task.id, task.title);
                          }}
                        >
                          Archive
                        </button>
                      </div>
                    </article>
                  </li>
                ))}
                </ol>
              </>
            )}
          </section>
        );
      })}

      <section className="panel panel-secondary board-side-panel">
        <div>
          <h2>Create task</h2>
          <p className="meta">
            This W7/W8 slice adds the first non-drag mutation path to the new frontend while keeping
            the board layout intentionally simple.
          </p>
          <p className="visually-hidden" aria-live="polite" role="status">
            {isBusy ? "Board update in progress." : actionError ?? actionStatus ?? ""}
          </p>
          <form
            className={`board-form${mutations.createTask.isPending ? " board-form-pending" : ""}`}
            onSubmit={handleCreateTask}
            noValidate
          >
            <label className="form-field" htmlFor="board-task-title">
              Title
            </label>
            <input
              id="board-task-title"
              className="text-input"
              ref={titleInputRef}
              value={title}
              onChange={(event) => {
                setTitle(event.target.value);
                if (fieldErrors.title) {
                  setFieldErrors((current) => ({ ...current, title: undefined }));
                }
              }}
              placeholder="Ship the next board slice"
              required
              aria-invalid={Boolean(fieldErrors.title)}
              aria-describedby={fieldErrors.title ? "board-task-title-error" : undefined}
            />
            {fieldErrors.title ? (
              <p id="board-task-title-error" className="form-error" role="alert">
                {fieldErrors.title}
              </p>
            ) : null}

            <div className="field-grid">
              <div>
                <label className="form-field" htmlFor="board-task-due">
                  Due date
                </label>
                <input
                  id="board-task-due"
                  className="text-input"
                  type="date"
                  ref={dueInputRef}
                  value={due}
                  onChange={(event) => {
                    setDue(event.target.value);
                    if (fieldErrors.due) {
                      setFieldErrors((current) => ({ ...current, due: undefined }));
                    }
                  }}
                  required
                  aria-invalid={Boolean(fieldErrors.due)}
                  aria-describedby={fieldErrors.due ? "board-task-due-error" : undefined}
                />
                {fieldErrors.due ? (
                  <p id="board-task-due-error" className="form-error" role="alert">
                    {fieldErrors.due}
                  </p>
                ) : null}
              </div>
              <div>
                <label className="form-field" htmlFor="board-task-priority">
                  Priority
                </label>
                <select
                  id="board-task-priority"
                  className="text-input"
                  value={priority}
                  onChange={(event) => setPriority(event.target.value as TaskPriority)}
                >
                  {priorityOptions.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <label className="form-field" htmlFor="board-task-note">
              Note
            </label>
            <textarea
              id="board-task-note"
              className="text-input text-area"
              value={note}
              onChange={(event) => setNote(event.target.value)}
              placeholder="Optional implementation note"
              rows={4}
            />
            <button
              className="nav-pill nav-pill-active auth-submit"
              type="submit"
              disabled={mutations.createTask.isPending}
            >
              {mutations.createTask.isPending ? "Creating..." : "Create task"}
            </button>
          </form>
        </div>

        <div>
          <h2>Archive Snapshot</h2>
          <p className="meta">
            Restore remains explicit and non-drag so the next frontend keeps a keyboard and touch
            fallback path while W8 is still maturing.
          </p>
          <p className="archive-total">{data.archived.length} archived cards</p>
          {data.archived.length === 0 ? (
            <p className="empty">Nothing is archived right now.</p>
          ) : (
            <div className="archive-list">
              {data.archived.map((task) => (
                <div
                  key={task.id}
                  className={`archive-item${pendingRestoreTaskID === task.id ? " archive-item-pending" : ""}`}
                >
                  <div>
                    <strong>{task.title}</strong>
                    <p className="meta">Return to {task.status}</p>
                  </div>
                  <button
                    className="action-button"
                    type="button"
                    disabled={pendingRestoreTaskID === task.id}
                    ref={(element) => setArchiveButtonRef(task.id, element)}
                    aria-label={`Restore ${task.title}`}
                    onClick={() => {
                      void handleRestoreTask(task.id, task.title, task.status);
                    }}
                  >
                    Restore
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function readErrorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }
  return "The board action failed.";
}
