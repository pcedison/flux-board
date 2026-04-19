import type { ArchivedTask } from "../../lib/api";
import { usePreferences } from "../../lib/preferences";

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
  const { copy, statusLabel } = usePreferences();
  const archivedLabel = copy.board.archiveCount(archived.length);

  return (
    <div>
      <h2>{copy.board.archiveTitle}</h2>
      <p className="meta">{copy.board.archiveHint}</p>
      <p className="archive-total">{archivedLabel}</p>
      {archived.length === 0 ? (
        <p className="empty">{copy.board.archiveEmpty}</p>
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
                <p className="meta">{copy.board.restoreTo(statusLabel(task.status))}</p>
              </div>
              <div className="archive-actions">
                <button
                  className="action-button"
                  type="button"
                  disabled={pendingRestoreTaskID === task.id || pendingDeleteArchivedTaskID === task.id}
                  ref={(element) => setArchiveButtonRef(task.id, element)}
                  aria-label={copy.board.restoreAria(task.title)}
                  onClick={() => {
                    void onRestoreTask(task.id, task.title, task.status);
                  }}
                >
                  {copy.board.restore}
                </button>
                <button
                  className="action-button action-button-secondary"
                  type="button"
                  disabled={pendingRestoreTaskID === task.id || pendingDeleteArchivedTaskID === task.id}
                  aria-label={copy.board.deleteAria(task.title)}
                  onClick={() => {
                    void onDeleteArchivedTask(task.id, task.title);
                  }}
                >
                  {copy.board.deletePermanently}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
