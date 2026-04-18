import type { FormEvent, RefObject } from "react";

import type { TaskPriority } from "../../lib/api";

type BoardComposerPanelProps = {
  due: string;
  dueInputRef: RefObject<HTMLInputElement | null>;
  fieldErrors: { due?: string; title?: string };
  isPending: boolean;
  note: string;
  onDueChange: (value: string) => void;
  onNoteChange: (value: string) => void;
  onPriorityChange: (value: TaskPriority) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onTitleChange: (value: string) => void;
  priority: TaskPriority;
  title: string;
  titleInputRef: RefObject<HTMLInputElement | null>;
};

const priorityOptions: TaskPriority[] = ["medium", "high", "critical"];

export function BoardComposerPanel({
  due,
  dueInputRef,
  fieldErrors,
  isPending,
  note,
  onDueChange,
  onNoteChange,
  onPriorityChange,
  onSubmit,
  onTitleChange,
  priority,
  title,
  titleInputRef,
}: BoardComposerPanelProps) {
  return (
    <div>
      <h2>New task</h2>
      <p className="meta">
        Add a task without leaving the board. Keyboard-friendly controls stay available whenever
        drag and drop is not the easiest option.
      </p>
      <form className={`board-form${isPending ? " board-form-pending" : ""}`} onSubmit={onSubmit} noValidate>
        <label className="form-field" htmlFor="board-task-title">
          Title
        </label>
        <input
          id="board-task-title"
          className="text-input"
          ref={titleInputRef}
          value={title}
          onChange={(event) => {
            onTitleChange(event.target.value);
          }}
          placeholder="Follow up with design review"
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
                onDueChange(event.target.value);
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
              onChange={(event) => onPriorityChange(event.target.value as TaskPriority)}
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
          onChange={(event) => {
            onNoteChange(event.target.value);
          }}
          placeholder="Add context, links, or handoff notes"
          rows={4}
        />
        <button className="nav-pill nav-pill-active auth-submit" type="submit" disabled={isPending}>
          {isPending ? "Creating..." : "Create task"}
        </button>
      </form>
    </div>
  );
}
