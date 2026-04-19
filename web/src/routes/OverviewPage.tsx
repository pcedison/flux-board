import { Link } from "react-router-dom";

import { QueryState } from "../components/QueryState";
import { useAppStatus } from "../lib/useAppStatus";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";
import { usePreferences } from "../lib/preferences";

export function OverviewPage() {
  const snapshot = useBoardSnapshot();
  const status = useAppStatus();
  const { copy } = usePreferences();

  return (
    <QueryState
      error={status.error ?? snapshot.error}
      errorTitle={copy.overview.errorTitle}
      isPending={status.isPending || snapshot.isPending}
      loadingMessage={copy.overview.loadingMessage}
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
  const { copy, formatDateTime, statusLabel } = usePreferences();
  const queued = snapshot.tasks.filter((task) => task.status === "queued").length;
  const active = snapshot.tasks.filter((task) => task.status === "active").length;
  const done = snapshot.tasks.filter((task) => task.status === "done").length;
  const statusHeading = status.status === "ready" ? copy.overview.ready : copy.overview.needsAttention;
  const retentionText =
    status.archiveRetentionDays === null
      ? copy.common.archiveRetentionIndefinite
      : copy.common.archiveRetentionDays(status.archiveRetentionDays);
  const primaryLink = status.needsSetup ? "/setup" : snapshot.session ? "/board" : "/login";
  const primaryLabel = status.needsSetup ? copy.overview.primarySetup : snapshot.session ? copy.common.openBoard : copy.overview.primarySignIn;
  const buildLabel = copy.common.version(status.version);
  const environmentLabel = copy.common.environment(status.environment);

  return (
    <div className="page-grid">
      <section className="panel">
        <h2>{copy.common.systemStatus}</h2>
        <p>{statusHeading}</p>
        <p className="meta">
          {`${buildLabel} in ${environmentLabel}. ${copy.common.generatedAt(formatDateTime(status.generatedAt))}`}
        </p>
        <ul className="checklist">
          {status.checks.map((check) => (
            <li key={check.name}>{check.message}</li>
          ))}
          <li>{retentionText}</li>
          <li>{copy.common.installedAt(status.runtimeOwnershipPath)}</li>
        </ul>
      </section>

      <section className="panel">
        <h2>{copy.common.boardSummary}</h2>
        <div className="stats-grid">
          <StatCard label={statusLabel("queued")} value={queued} />
          <StatCard label={statusLabel("active")} value={active} />
          <StatCard label={statusLabel("done")} value={done} />
          <StatCard label={copy.laneLabels.archived} value={snapshot.archived.length} />
        </div>
        <p className="meta">
          {snapshot.session
            ? copy.common.sessionExpiresAt(formatDateTime(snapshot.session.expiresAt))
            : copy.common.sessionSignedOut}
        </p>
      </section>

      <section className="panel">
        <h2>{copy.overview.nextSteps}</h2>
        <ul className="checklist">
          <li>{status.needsSetup ? copy.overview.setupNeeded : copy.overview.setupReady}</li>
          <li>{copy.overview.cleanupSummary(status.sessionCleanupEvery, status.archiveCleanupEvery)}</li>
          <li>{copy.overview.rollbackSummary(status.legacyRollbackPath)}</li>
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
