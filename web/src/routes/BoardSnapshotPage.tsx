import type { TaskStatus } from "../lib/api";
import { QueryState } from "../components/QueryState";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

const lanes: Array<{ label: string; status: TaskStatus }> = [
  { label: "Queued", status: "queued" },
  { label: "Active", status: "active" },
  { label: "Done", status: "done" },
];

export function BoardSnapshotPage() {
  const snapshot = useBoardSnapshot();

  return (
    <QueryState
      error={snapshot.error}
      errorTitle="Failed to load board snapshot"
      isPending={snapshot.isPending}
      loadingMessage="Loading the current lane breakdown from the Go API."
    >
      {snapshot.data ? <BoardSnapshotContent data={snapshot.data} /> : null}
    </QueryState>
  );
}

function BoardSnapshotContent({
  data,
}: {
  data: NonNullable<ReturnType<typeof useBoardSnapshot>["data"]>;
}) {
  return (
    <div className="board-grid">
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
              <div className="lane-list">
                {tasks.map((task) => (
                  <article key={task.id} className="card">
                    <div className="card-row">
                      <strong>{task.title}</strong>
                      <span className={`priority priority-${task.priority}`}>{task.priority}</span>
                    </div>
                    <p className="meta">
                      Due {task.due} · order {task.sort_order}
                    </p>
                    {task.note ? <p className="card-note">{task.note}</p> : null}
                  </article>
                ))}
              </div>
            )}
          </section>
        );
      })}

      <section className="panel panel-secondary">
        <h2>Archive Snapshot</h2>
        <p className="meta">
          Archived cards are still managed by the current embedded UI and smoke coverage. This shell
          shows the current count while W8 remains focused on the live board experience first.
        </p>
        <p className="archive-total">{data.archived.length} archived cards</p>
      </section>
    </div>
  );
}
