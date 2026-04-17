import type { ArchivedTask } from "../../lib/api";

type BoardArchivePanelProps = {
  archived: ArchivedTask[];
  onRestoreTask: (id: string, taskTitle: string, status: ArchivedTask["status"]) => void;
  pendingRestoreTaskID: string | null;
  setArchiveButtonRef: (id: string, element: HTMLButtonElement | null) => void;
};

export function BoardArchivePanel({
  archived,
  onRestoreTask,
  pendingRestoreTaskID,
  setArchiveButtonRef,
}: BoardArchivePanelProps) {
  return (
    <div>
      <h2>Archive Snapshot</h2>
      <p className="meta">
        Restore remains explicit and non-drag so the next frontend keeps a keyboard and touch
        fallback path while W8 is still maturing.
      </p>
      <p className="archive-total">{archived.length} archived cards</p>
      {archived.length === 0 ? (
        <p className="empty">Nothing is archived right now.</p>
      ) : (
        <div className="archive-list">
          {archived.map((task) => (
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
                  void onRestoreTask(task.id, task.title, task.status);
                }}
              >
                Restore
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
