import { Link } from "react-router-dom";

import { QueryState } from "../components/QueryState";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

export function OverviewPage() {
  const snapshot = useBoardSnapshot();

  return (
    <QueryState
      error={snapshot.error}
      errorTitle="Failed to load frontend foundation snapshot"
      isPending={snapshot.isPending}
      loadingMessage="Reading the current auth and board snapshot from the Go API."
    >
      {snapshot.data ? <OverviewContent data={snapshot.data} /> : null}
    </QueryState>
  );
}

function OverviewContent({ data }: { data: NonNullable<ReturnType<typeof useBoardSnapshot>["data"]> }) {
  const queued = data.tasks.filter((task) => task.status === "queued").length;
  const active = data.tasks.filter((task) => task.status === "active").length;
  const done = data.tasks.filter((task) => task.status === "done").length;

  return (
    <div className="page-grid">
      <section className="panel">
        <h2>Session</h2>
        <p>{data.session ? "Authenticated session detected" : "No active session detected"}</p>
        <p className="meta">
          {data.session
            ? `Expires at ${new Date(data.session.expiresAt).toLocaleString()}`
            : "Use the new /login route to establish the shared session cookie before opening the guarded board snapshot."}
        </p>
      </section>

      <section className="panel">
        <h2>Board Totals</h2>
        <div className="stats-grid">
          <StatCard label="Queued" value={queued} />
          <StatCard label="Active" value={active} />
          <StatCard label="Done" value={done} />
          <StatCard label="Archived" value={data.archived.length} />
        </div>
      </section>

      <section className="panel">
        <h2>Why this slice exists</h2>
        <ul className="checklist">
          <li>Keep the future frontend isolated from the embedded MVP runtime.</li>
          <li>Establish build and typecheck gates before interactive board work starts.</li>
          <li>Read real API data now so later mutation work has a stable transport seam.</li>
        </ul>
        <div className="action-row">
          <Link className="nav-pill nav-pill-active" to={data.session ? "/board" : "/login"}>
            {data.session ? "Open guarded board" : "Open sign-in route"}
          </Link>
        </div>
      </section>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="stat-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
