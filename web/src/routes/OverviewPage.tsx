import { Link } from "react-router-dom";

import { QueryState } from "../components/QueryState";
import { useAppStatus } from "../lib/useAppStatus";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

export function OverviewPage() {
  const snapshot = useBoardSnapshot();
  const status = useAppStatus();

  return (
    <QueryState
      error={status.error ?? snapshot.error}
      errorTitle="Unable to load board overview"
      isPending={status.isPending || snapshot.isPending}
      loadingMessage="Loading your board summary and app status."
    >
      {status.data && snapshot.data ? <OverviewContent snapshot={snapshot.data} status={status.data} /> : null}
    </QueryState>
  );
}

function OverviewContent({
  snapshot,
  status,
}: {
  snapshot: NonNullable<ReturnType<typeof useBoardSnapshot>["data"]>;
  status: NonNullable<ReturnType<typeof useAppStatus>["data"]>;
}) {
  const queued = snapshot.tasks.filter((task) => task.status === "queued").length;
  const active = snapshot.tasks.filter((task) => task.status === "active").length;
  const done = snapshot.tasks.filter((task) => task.status === "done").length;
  const statusHeading = status.status === "ready" ? "Ready to use" : "Needs attention";
  const retentionText =
    status.archiveRetentionDays === null
      ? "Archived cards stay until you remove them manually."
      : `Archived cards auto-delete after ${status.archiveRetentionDays} days.`;
  const primaryLink = status.needsSetup ? "/setup" : snapshot.session ? "/board" : "/login";
  const primaryLabel = status.needsSetup ? "Open setup" : snapshot.session ? "Open board" : "Open sign-in";
  const buildLabel = status.version === "dev" ? "Unreleased build" : `Version ${status.version}`;
  const environmentLabel = status.environment === "development" ? "local environment" : `${status.environment} environment`;

  return (
    <div className="page-grid">
      <section className="panel">
        <h2>App status</h2>
        <p>{statusHeading}</p>
        <p className="meta">
          {`${buildLabel} in ${environmentLabel}. Updated ${new Date(status.generatedAt).toLocaleString()}.`}
        </p>
        <ul className="checklist">
          {status.checks.map((check) => (
            <li key={check.name}>{check.message}</li>
          ))}
          <li>{retentionText}</li>
          <li>{`Installed at ${status.runtimeOwnershipPath}.`}</li>
        </ul>
      </section>

      <section className="panel">
        <h2>Board summary</h2>
        <div className="stats-grid">
          <StatCard label="Queued" value={queued} />
          <StatCard label="Active" value={active} />
          <StatCard label="Done" value={done} />
          <StatCard label="Archived" value={snapshot.archived.length} />
        </div>
        <p className="meta">
          {snapshot.session
            ? `Signed in on this browser until ${new Date(snapshot.session.expiresAt).toLocaleString()}.`
            : "This browser is currently signed out."}
        </p>
      </section>

      <section className="panel">
        <h2>Next steps</h2>
        <ul className="checklist">
          <li>{status.needsSetup ? "Finish setup to create the board password." : "Setup is complete and the board is ready to use."}</li>
          <li>{`Expired sessions are cleared every ${status.sessionCleanupEvery}, and archived cards are checked every ${status.archiveCleanupEvery}.`}</li>
          <li>{`Need to roll back? The previous app path is still available at ${status.legacyRollbackPath}.`}</li>
        </ul>
        <div className="action-row">
          <Link className="nav-pill nav-pill-active" to={primaryLink}>
            {primaryLabel}
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
