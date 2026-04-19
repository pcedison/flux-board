import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { RefCallback } from "react";

import type { Task, TaskStatus } from "../../lib/api";
import { usePreferences } from "../../lib/usePreferences";

type BoardTaskCardProps = {
  index: number;
  isActive: boolean;
  isBusy: boolean;
  isSelected: boolean;
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

type TaskCardBodyProps = {
  dueLabel: string;
  note: string;
  priority: Task["priority"];
  priorityText: string;
  title: string;
  isOverdue: boolean;
};

const cardTransition = "box-shadow 180ms ease, border-color 180ms ease, background-color 180ms ease, opacity 160ms ease";

function computeIsOverdue(due: string): boolean {
  return due ? new Date(due + 'T00:00:00') < new Date(new Date().toDateString()) : false;
}

export function BoardTaskCard({
  index,
  isActive,
  isBusy,
  isSelected,
  laneStatus,
  onCardFocus,
  onCardNavigate,
  onSelectTask,
  setRef,
  task,
}: BoardTaskCardProps) {
  const { copy, formatDate, priorityLabel } = usePreferences();
  const { isDragging, listeners, setNodeRef, transform, transition } = useSortable({
    disabled: isBusy,
    id: task.id,
    transition: {
      duration: 180,
      easing: "cubic-bezier(0.2, 0, 0, 1)",
    },
  });

  const cardStyle = {
    transform: CSS.Transform.toString(transform),
    transition: transition ? `${transition}, ${cardTransition}` : cardTransition,
  };

  const setCardNodeRef: RefCallback<HTMLElement> = (element) => {
    setNodeRef(element);
    setRef(element);
  };

  const isOverdue = computeIsOverdue(task.due);

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
      {...listeners}
    >
      <TaskCardBody
        dueLabel={copy.board.due(formatDate(task.due))}
        isOverdue={isOverdue}
        note={task.note}
        priority={task.priority}
        priorityText={priorityLabel(task.priority)}
        title={task.title}
      />
    </article>
  );
}

export function BoardTaskCardPreview({ task }: { task: Task }) {
  const { copy, formatDate, priorityLabel } = usePreferences();
  const isOverdue = computeIsOverdue(task.due);

  return (
    <article className="card card-overlay" aria-hidden="true">
      <TaskCardBody
        dueLabel={copy.board.due(formatDate(task.due))}
        isOverdue={isOverdue}
        note={task.note}
        priority={task.priority}
        priorityText={priorityLabel(task.priority)}
        title={task.title}
      />
    </article>
  );
}

function TaskCardBody({ dueLabel, note, priority, priorityText, title, isOverdue }: TaskCardBodyProps) {
  return (
    <>
      <div className="card-row">
        <strong>{title}</strong>
      </div>
      <div className="card-row card-meta-row">
        <span className={`priority priority-${priority}`}>{priorityText}</span>
        <p className={`meta${isOverdue ? ' overdue' : ''}`}>{dueLabel}</p>
      </div>
      {note ? <p className="card-note">{note}</p> : null}
    </>
  );
}
