import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { RefCallback } from "react";

import type { Task, TaskStatus } from "../../lib/api";
import type { MoveTarget, MoveTaskRequest } from "./types";

const moveTargets: Record<TaskStatus, MoveTarget[]> = {
  queued: [{ label: "Move to Active", status: "active" }],
  active: [
    { label: "Back to Queued", status: "queued" },
    { label: "Move to Done", status: "done" },
  ],
  done: [{ label: "Back to Active", status: "active" }],
};

type BoardTaskCardProps = {
  index: number;
  isActive: boolean;
  isBusy: boolean;
  laneLabel: string;
  laneStatus: TaskStatus;
  onCardFocus: (taskId: string) => void;
  onCardNavigate: (
    taskId: string,
    laneStatus: TaskStatus,
    index: number,
    key: "ArrowUp" | "ArrowDown" | "ArrowLeft" | "ArrowRight",
  ) => void;
  onArchiveTask: (id: string, taskTitle: string) => void;
  onEditTask: (task: Task) => void;
  onMoveTask: (move: MoveTaskRequest, announcement: string) => void;
  setRef: RefCallback<HTMLElement>;
  task: Task;
  tasks: Task[];
};

export function BoardTaskCard({
  index,
  isActive,
  isBusy,
  laneLabel,
  laneStatus,
  onCardFocus,
  onCardNavigate,
  onArchiveTask,
  onEditTask,
  onMoveTask,
  setRef,
  task,
  tasks,
}: BoardTaskCardProps) {
  const { attributes, isDragging, listeners, setActivatorNodeRef, setNodeRef, transform, transition } = useSortable({
    id: task.id,
  });

  const cardStyle = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const setCardNodeRef: RefCallback<HTMLElement> = (element) => {
    setNodeRef(element);
    setRef(element);
  };

  return (
    <article
      className={`card${isBusy ? " card-pending" : ""}${isDragging ? " card-dragging" : ""}`}
      ref={setCardNodeRef}
      onFocus={() => {
        onCardFocus(task.id);
      }}
      onKeyDown={(event) => {
        if (event.target !== event.currentTarget) {
          return;
        }

        if (
          event.key === "ArrowUp" ||
          event.key === "ArrowDown" ||
          event.key === "ArrowLeft" ||
          event.key === "ArrowRight"
        ) {
          event.preventDefault();
          onCardNavigate(task.id, laneStatus, index, event.key as "ArrowUp" | "ArrowDown" | "ArrowLeft" | "ArrowRight");
        }
      }}
      style={cardStyle}
      tabIndex={isActive ? 0 : -1}
    >
      <div className="card-row">
        <div className="card-row-main">
          <button
            className="drag-handle"
            type="button"
            disabled={isBusy}
            aria-label={`Drag ${task.title} to reorder within ${laneLabel}`}
            ref={setActivatorNodeRef}
            {...attributes}
            {...listeners}
            tabIndex={-1}
          >
            Drag
          </button>
          <strong>{task.title}</strong>
        </div>
        <span className={`priority priority-${task.priority}`}>{task.priority}</span>
      </div>
      <p className="meta">
        Due {task.due}
      </p>
      {task.note ? <p className="card-note">{task.note}</p> : null}
      <div className="card-actions">
        {index > 0 ? (
          <button
            className="action-button"
            type="button"
            disabled={isBusy}
            aria-label={`Move ${task.title} up within ${laneLabel}`}
            onClick={() => {
              const previousTask = tasks[index - 1];
              void onMoveTask(
                {
                  id: task.id,
                  status: task.status,
                  anchorTaskId: previousTask.id,
                  placeAfter: false,
                },
                `Moved ${task.title} up within ${laneLabel}.`,
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
            disabled={isBusy}
            aria-label={`Move ${task.title} down within ${laneLabel}`}
            onClick={() => {
              const nextTask = tasks[index + 1];
              void onMoveTask(
                {
                  id: task.id,
                  status: task.status,
                  anchorTaskId: nextTask.id,
                  placeAfter: true,
                },
                `Moved ${task.title} down within ${laneLabel}.`,
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
            disabled={isBusy}
            aria-label={`${target.label} (${task.title})`}
            onClick={() => {
              const targetLabel = target.label.replace("Move to ", "").replace("Back to ", "");
              void onMoveTask(
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
          className="action-button"
          type="button"
          disabled={isBusy}
          aria-label={`Edit ${task.title}`}
          onClick={() => {
            onEditTask(task);
          }}
        >
          Edit
        </button>
        <button
          className="action-button action-button-secondary"
          type="button"
          disabled={isBusy}
          aria-label={`Archive ${task.title}`}
          onClick={() => {
            void onArchiveTask(task.id, task.title);
          }}
        >
          Archive
        </button>
      </div>
    </article>
  );
}
