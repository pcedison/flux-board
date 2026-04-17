import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";

import type { Task } from "../../lib/api";
import type { BoardLaneDescriptor, MoveTaskRequest } from "./types";
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
  onArchiveTask: (id: string, taskTitle: string) => void;
  onMoveTask: (move: MoveTaskRequest, announcement: string) => void;
  setCardRef: (id: string, element: HTMLElement | null) => void;
  tasks: Task[];
};

export function BoardLane({
  activeCardId = null,
  isTaskBusy,
  lane,
  onCardNavigate = () => {},
  onCardFocus = () => {},
  onArchiveTask,
  onMoveTask,
  setCardRef,
  tasks,
}: BoardLaneProps) {
  return (
    <section className="lane" aria-labelledby={`lane-${lane.status}`}>
      <div className="lane-head">
        <h2 id={`lane-${lane.status}`}>{lane.label}</h2>
        <span>{tasks.length}</span>
      </div>

      {tasks.length === 0 ? (
        <p className="empty">No tasks in this lane yet.</p>
      ) : (
        <SortableContext items={tasks.map((task) => task.id)} strategy={verticalListSortingStrategy}>
          <p id={`lane-${lane.status}-hint`} className="visually-hidden">
            Use Move up or Move down to reorder cards within the {lane.label} lane.
          </p>
          <p id={`lane-${lane.status}-nav`} className="visually-hidden">
            Use Tab to reach one card shell, Arrow keys to move between cards, and Tab again to reach the action
            buttons.
          </p>
          <ol className="lane-list" aria-describedby={`lane-${lane.status}-hint lane-${lane.status}-nav`}>
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
                  laneLabel={lane.label}
                  laneStatus={lane.status}
                  onCardFocus={onCardFocus}
                  onCardNavigate={onCardNavigate}
                  onArchiveTask={onArchiveTask}
                  onMoveTask={onMoveTask}
                  setRef={(element) => setCardRef(task.id, element)}
                  task={task}
                  tasks={tasks}
                />
              </li>
            ))}
          </ol>
        </SortableContext>
      )}
    </section>
  );
}
