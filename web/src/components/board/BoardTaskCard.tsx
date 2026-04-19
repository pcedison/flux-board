import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { RefCallback } from "react";

import type { Task, TaskStatus } from "../../lib/api";
import { usePreferences } from "../../lib/preferences";

type BoardTaskCardProps = {
  index: number;
  isActive: boolean;
  isBusy: boolean;
  isSelected: boolean;
  laneLabel: string;
  laneStatus: TaskStatus;
  onCardFocus: (taskId: string) => void;
  onCardNavigate: (
    taskId: string,
    laneStatus: TaskStatus,
    index: number,
    key: "ArrowUp" | "ArrowDown" | "ArrowLeft" | "ArrowRight",
  ) => void;
  onSelectTask: (task: Task) => void;
  setRef: RefCallback<HTMLElement>;
  task: Task;
};

export function BoardTaskCard({
  index,
  isActive,
  isBusy,
  isSelected,
  laneLabel,
  laneStatus,
  onCardFocus,
  onCardNavigate,
  onSelectTask,
  setRef,
  task,
}: BoardTaskCardProps) {
  const { copy, formatDate, priorityLabel } = usePreferences();
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
      className={`card${isBusy ? " card-pending" : ""}${isDragging ? " card-dragging" : ""}${isSelected ? " card-selected" : ""}`}
      ref={setCardNodeRef}
      onFocus={() => {
        onCardFocus(task.id);
      }}
      onClick={() => {
        onSelectTask(task);
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
        <strong>{task.title}</strong>
        <button
          className="drag-handle"
          type="button"
          disabled={isBusy}
          aria-label={copy.board.dragLabel(task.title, laneLabel)}
          ref={setActivatorNodeRef}
          {...attributes}
          {...listeners}
        >
          {copy.board.dragAction}
        </button>
      </div>
      <div className="card-row card-meta-row">
        <span className={`priority priority-${task.priority}`}>{priorityLabel(task.priority)}</span>
        <p className="meta">{copy.board.due(formatDate(task.due))}</p>
      </div>
      {task.note ? <p className="card-note">{task.note}</p> : null}
    </article>
  );
}
