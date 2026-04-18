import type { ArchivedTask } from "../../lib/api";

type BoardArchivePanelProps = {
  archived: ArchivedTask[];
  onDeleteArchivedTask: (id: string, taskTitle: string) => void;
  onRestoreTask: (id: string, taskTitle: string, status: ArchivedTask["status"]) => void;
  pendingDeleteArchivedTaskID: string | null;
  pendingRestoreTaskID: string | null;
  setArchiveButtonRef: (id: string, element: HTMLButtonElement | null) => void;
};

export function BoardArchivePanel({
  archived,
  onDeleteArchivedTask,
  onRestoreTask,
  pendingDeleteArchivedTaskID,
  pendingRestoreTaskID,
  setArchiveButtonRef,
}: BoardArchivePanelProps) {
  const archivedLabel = `${archived.length} archived ${archived.length === 1 ? "task" : "tasks"}`;

  return (
    <div>
      <h2>Archive</h2>
      <p className="meta">Restore archived tasks or remove them for good.</p>
      <p className="archive-total">{archivedLabel}</p>
      {archived.length === 0 ? (
        <p className="empty">Nothing is archived right now.</p>
      ) : (
        <div className="archive-list">
          {archived.map((task) => (
            <div
              key={task.id}
              className={`archive-item${
                pendingRestoreTaskID === task.id || pendingDeleteArchivedTaskID === task.id ? " archive-item-pending" : ""
              }`}
            >
              <div>
                <strong>{task.title}</strong>
                <p className="meta">Return to {task.status}</p>
              </div>
              <div className="archive-actions">
                <button
                  className="action-button"
                  type="button"
                  disabled={pendingRestoreTaskID === task.id || pendingDeleteArchivedTaskID === task.id}
                  ref={(element) => setArchiveButtonRef(task.id, element)}
                  aria-label={`Restore ${task.title}`}
                  onClick={() => {
                    void onRestoreTask(task.id, task.title, task.status);
                  }}
                >
                  Restore
                </button>
                <button
                  className="action-button action-button-secondary"
                  type="button"
                  disabled={pendingRestoreTaskID === task.id || pendingDeleteArchivedTaskID === task.id}
                  aria-label={`Delete ${task.title} permanently`}
                  onClick={() => {
                    void onDeleteArchivedTask(task.id, task.title);
                  }}
                >
                  Delete permanently
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
