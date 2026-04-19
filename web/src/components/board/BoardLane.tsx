import { useDroppable } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useEffect, useState } from "react";

import type { Task } from "../../lib/api";
import { usePreferences } from "../../lib/preferences";
import { getLaneDropId } from "./dragAndDrop";
import type { BoardLaneDescriptor } from "./types";
import { BoardTaskCard } from "./BoardTaskCard";

type BoardLaneProps = {
  activeCardId?: string | null;
  isTaskBusy: (id: string) => boolean;
  lane: BoardLaneDescriptor;
  onCardNavigate?: (
    taskId: string,
    laneStatus: BoardLaneDescriptor["status"],
    index: number,
    key: "ArrowUp" | "ArrowDown" | "ArrowLeft" | "ArrowRight",
  ) => void;
  onCardFocus?: (taskId: string) => void;
  onSelectTask: (task: Task) => void;
  selectedTaskId?: string | null;
  setCardRef: (id: string, element: HTMLElement | null) => void;
  tasks: Task[];
};

export function BoardLane({
  activeCardId = null,
  isTaskBusy,
  lane,
  onCardNavigate = () => {},
  onCardFocus = () => {},
  onSelectTask,
  selectedTaskId = null,
  setCardRef,
  tasks,
}: BoardLaneProps) {
  const { copy } = usePreferences();
  const { isOver, setNodeRef } = useDroppable({
    id: getLaneDropId(lane.status),
  });
  const [isCollapsed, setIsCollapsed] = useState(tasks.length > 1);
  const canCollapse = tasks.length > 1;

  useEffect(() => {
    if (tasks.length <= 1) {
      setIsCollapsed(false);
    }
  }, [tasks.length]);

  return (
    <section
      className={`lane${isOver ? " lane-over" : ""}${canCollapse && isCollapsed ? " lane-collapsed" : ""}`}
      ref={setNodeRef}
      aria-labelledby={`lane-${lane.status}`}
    >
      <div className="lane-head">
        <h2 id={`lane-${lane.status}`}>{lane.label}</h2>
        <span>{tasks.length}</span>
      </div>

      {tasks.length === 0 ? (
        <p className="empty">{copy.board.laneEmpty}</p>
      ) : (
        <SortableContext items={tasks.map((task) => task.id)} strategy={verticalListSortingStrategy}>
          <p id={`lane-${lane.status}-nav`} className="visually-hidden">
            {copy.board.laneNavigationHint}
          </p>
          {canCollapse ? (
            <button
              className={`lane-toggle${isCollapsed ? " lane-toggle-collapsed" : ""}`}
              type="button"
              aria-expanded={!isCollapsed}
              onClick={() => setIsCollapsed((current) => !current)}
            >
              <span className="lane-toggle-title">{copy.board.laneHiddenSummary(tasks.length)}</span>
              <span className="lane-toggle-hint">
                {isCollapsed ? copy.board.laneExpandHint : copy.board.laneCollapseHint}
              </span>
            </button>
          ) : null}
          {!isCollapsed ? (
            <ol className="lane-list" aria-describedby={`lane-${lane.status}-nav`}>
              {tasks.map((task, index) => (
                <li
                  key={task.id}
                  className="lane-list-item"
                  aria-posinset={index + 1}
                  aria-setsize={tasks.length}
                >
                  <BoardTaskCard
                    index={index}
                    isBusy={isTaskBusy(task.id)}
                    isActive={activeCardId === task.id}
                    isSelected={selectedTaskId === task.id}
                    laneStatus={lane.status}
                    onCardFocus={onCardFocus}
                    onCardNavigate={onCardNavigate}
                    onSelectTask={onSelectTask}
                    setRef={(element) => setCardRef(task.id, element)}
                    task={task}
                  />
                </li>
              ))}
            </ol>
          ) : null}
        </SortableContext>
      )}
    </section>
  );
}
