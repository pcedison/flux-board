import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, type FormEvent, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { QueryState } from "../components/QueryState";
import { useAppStatus } from "../lib/useAppStatus";
import { useBoardSnapshot, boardSnapshotQueryKey } from "../lib/useBoardSnapshot";
import {
  changePassword,
  exportBoardData,
  fetchSessions,
  fetchSettings,
  importBoardData,
  revokeSession,
  updateSettings,
  type AppSettings,
  type ExportBundle,
} from "../lib/api";
import { clearAuthSessionData } from "../lib/useAuthSession";
import { type AppLocale, type AppTheme } from "../lib/preferences";
import { usePreferences } from "../lib/usePreferences";

const settingsQueryKey = ["settings"] as const;
const sessionsQueryKey = ["settings-sessions"] as const;

export function SettingsPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { copy, formatDateTime, locale, setLocale, setTheme, theme, statusLabel } = usePreferences();
  const settings = useQuery({ queryKey: settingsQueryKey, queryFn: fetchSettings });
  const sessions = useQuery({ queryKey: sessionsQueryKey, queryFn: fetchSessions });
  const appStatus = useAppStatus();
  const snapshot = useBoardSnapshot();

  const [draftArchiveRetentionEnabled, setDraftArchiveRetentionEnabled] = useState<boolean | null>(null);
  const [draftArchiveRetentionDays, setDraftArchiveRetentionDays] = useState<string | null>(null);
  const [passwordStatus, setPasswordStatus] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [settingsStatus, setSettingsStatus] = useState<string | null>(null);
  const [settingsError, setSettingsError] = useState<string | null>(null);
  const [importStatus, setImportStatus] = useState<string | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const archiveRetentionEnabled =
    draftArchiveRetentionEnabled ?? (settings.data ? settings.data.archiveRetentionDays !== null : false);
  const archiveRetentionDays =
    draftArchiveRetentionDays ?? String(settings.data?.archiveRetentionDays ?? 30);

  const updateSettingsMutation = useMutation({
    mutationFn: (input: AppSettings) => updateSettings(input),
    onSuccess: async (nextSettings) => {
      queryClient.setQueryData(settingsQueryKey, nextSettings);
      await queryClient.invalidateQueries({ queryKey: boardSnapshotQueryKey });
      setDraftArchiveRetentionEnabled(nextSettings.archiveRetentionDays !== null);
      setDraftArchiveRetentionDays(String(nextSettings.archiveRetentionDays ?? 30));
      setSettingsError(null);
      setSettingsStatus(
        nextSettings.archiveRetentionDays === null
          ? copy.settings.archiveSavedForever
          : copy.settings.archiveSavedDays(nextSettings.archiveRetentionDays),
      );
    },
    onError: (error) => {
      setSettingsStatus(null);
      setSettingsError(error instanceof Error ? error.message : copy.settings.settingsSaveFailed);
    },
  });

  const changePasswordMutation = useMutation({
    mutationFn: ({ currentPassword, newPassword }: { currentPassword: string; newPassword: string }) =>
      changePassword(currentPassword, newPassword),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: sessionsQueryKey });
      setPasswordError(null);
      setPasswordStatus(copy.settings.passwordUpdated);
    },
    onError: (error) => {
      setPasswordStatus(null);
      setPasswordError(error instanceof Error ? error.message : copy.settings.passwordUpdateFailed);
    },
  });

  const revokeSessionMutation = useMutation({
    mutationFn: revokeSession,
    onSuccess: async (_, token) => {
      await queryClient.invalidateQueries({ queryKey: sessionsQueryKey });
      if (sessions.data?.find((session) => session.token === token)?.current) {
        clearAuthSessionData(queryClient);
        navigate("/login", { replace: true });
      }
    },
  });

  const exportMutation = useMutation({
    mutationFn: exportBoardData,
    onSuccess: (bundle) => {
      downloadExport(bundle);
      setImportError(null);
      setImportStatus(copy.settings.importDownloaded);
    },
    onError: (error) => {
      setImportStatus(null);
      setImportError(error instanceof Error ? error.message : copy.settings.importDownloadFailed);
    },
  });

  const importMutation = useMutation({
    mutationFn: importBoardData,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: boardSnapshotQueryKey }),
        queryClient.invalidateQueries({ queryKey: settingsQueryKey }),
      ]);
      setImportError(null);
      setImportStatus(copy.settings.importRestored);
    },
    onError: (error) => {
      setImportStatus(null);
      setImportError(error instanceof Error ? error.message : copy.settings.importRestoreFailed);
    },
  });

  return (
    <QueryState
      error={settings.error ?? sessions.error}
      errorTitle={copy.settings.errorTitle}
      isPending={settings.isPending || sessions.isPending}
      loadingMessage={copy.settings.loadingMessage}
    >
      {settings.data && sessions.data ? (
        <SettingsContent
          appStatus={appStatus}
          archiveRetentionDays={archiveRetentionDays}
          archiveRetentionEnabled={archiveRetentionEnabled}
          changePasswordMutation={changePasswordMutation}
          exportMutation={exportMutation}
          formatDateTime={formatDateTime}
          importError={importError}
          importMutation={importMutation}
          importStatus={importStatus}
          locale={locale}
          passwordError={passwordError}
          passwordStatus={passwordStatus}
          revokeSessionMutation={revokeSessionMutation}
          sessions={sessions.data}
          setArchiveRetentionDays={setDraftArchiveRetentionDays}
          setArchiveRetentionEnabled={setDraftArchiveRetentionEnabled}
          setLocale={setLocale}
          setTheme={setTheme}
          settingsError={settingsError}
          settingsMutation={updateSettingsMutation}
          settingsStatus={settingsStatus}
          snapshot={snapshot}
          statusLabel={statusLabel}
          theme={theme}
        />
      ) : null}
    </QueryState>
  );
}

function SettingsContent({
  appStatus,
  archiveRetentionDays,
  archiveRetentionEnabled,
  changePasswordMutation,
  exportMutation,
  formatDateTime,
  importError,
  importMutation,
  importStatus,
  locale,
  passwordError,
  passwordStatus,
  revokeSessionMutation,
  sessions,
  setArchiveRetentionDays,
  setArchiveRetentionEnabled,
  setLocale,
  setTheme,
  settingsError,
  settingsMutation,
  settingsStatus,
  snapshot,
  statusLabel,
  theme,
}: {
  appStatus: ReturnType<typeof useAppStatus>;
  archiveRetentionDays: string;
  archiveRetentionEnabled: boolean;
  changePasswordMutation: ReturnType<typeof useMutation<void, Error, { currentPassword: string; newPassword: string }>>;
  exportMutation: ReturnType<typeof useMutation<ExportBundle, Error, void>>;
  formatDateTime: (value: number) => string;
  importError: string | null;
  importMutation: ReturnType<typeof useMutation<void, Error, ExportBundle>>;
  importStatus: string | null;
  locale: AppLocale;
  passwordError: string | null;
  passwordStatus: string | null;
  revokeSessionMutation: ReturnType<typeof useMutation<void, Error, string>>;
  sessions: Awaited<ReturnType<typeof fetchSessions>>;
  setArchiveRetentionDays: (value: string) => void;
  setArchiveRetentionEnabled: (value: boolean) => void;
  setLocale: (locale: AppLocale) => void;
  setTheme: (theme: AppTheme) => void;
  settingsError: string | null;
  settingsMutation: ReturnType<typeof useMutation<AppSettings, Error, AppSettings>>;
  settingsStatus: string | null;
  snapshot: ReturnType<typeof useBoardSnapshot>;
  statusLabel: (value: "queued" | "active" | "done") => string;
  theme: AppTheme;
}) {
  const { copy } = usePreferences();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const activeSessionsSummary = useMemo(
    () => copy.settings.sessionsSummary(sessions.length),
    [copy.settings, sessions.length],
  );

  async function handleSettingsSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    settingsMutation.reset();
    const nextSettings: AppSettings = {
      archiveRetentionDays: archiveRetentionEnabled ? Number(archiveRetentionDays) : null,
    };
    await settingsMutation.mutateAsync(nextSettings);
  }

  async function handlePasswordSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (newPassword !== confirmPassword) {
      return;
    }
    await changePasswordMutation.mutateAsync({ currentPassword, newPassword });
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
  }

  async function handleImportFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    try {
      const parsed = JSON.parse(await file.text()) as ExportBundle;
      await importMutation.mutateAsync(parsed);
    } finally {
      event.target.value = "";
    }
  }

  const boardCounts = snapshot.data
    ? [
        { label: statusLabel("queued"), value: snapshot.data.tasks.filter((task) => task.status === "queued").length },
        { label: statusLabel("active"), value: snapshot.data.tasks.filter((task) => task.status === "active").length },
        { label: statusLabel("done"), value: snapshot.data.tasks.filter((task) => task.status === "done").length },
        { label: copy.settings.overviewArchivedCountLabel, value: snapshot.data.archived.length },
      ]
    : [];

  return (
    <div className="page-grid settings-grid">
      <section className="panel">
        <h2>{copy.settings.appearanceTitle}</h2>
        <p className="meta">{copy.settings.appearanceHint}</p>
        <div className="board-form">
          <div className="field-grid">
            <div>
              <label className="form-field" htmlFor="settings-locale">
                {copy.common.language}
              </label>
              <select
                id="settings-locale"
                className="text-input"
                value={locale}
                onChange={(event) => setLocale(event.target.value as AppLocale)}
              >
                <option value="zh-TW">繁體中文</option>
                <option value="en">English</option>
              </select>
            </div>
            <div>
              <label className="form-field" htmlFor="settings-theme">
                {copy.common.theme}
              </label>
              <select
                id="settings-theme"
                className="text-input"
                value={theme}
                onChange={(event) => setTheme(event.target.value as AppTheme)}
              >
                <option value="light">{copy.common.light}</option>
                <option value="dark">{copy.common.dark}</option>
              </select>
            </div>
          </div>
        </div>
      </section>

      <section className="panel">
        <h2>{copy.common.systemStatus}</h2>
        {appStatus.isPending ? (
          <p className="meta">{copy.common.loading}</p>
        ) : appStatus.error ? (
          <p className="form-error">{appStatus.error.message}</p>
        ) : appStatus.data ? (
          <>
            <p>
              {appStatus.data.status === "ready" ? copy.settings.overviewStateReady : copy.settings.overviewStateAttention}
            </p>
            <p className="meta">
              {copy.settings.overviewRuntime(
                copy.common.version(appStatus.data.version),
                copy.common.environment(appStatus.data.environment),
              )}
            </p>
            <p className="meta">{copy.settings.overviewStatusUpdated(formatDateTime(appStatus.data.generatedAt))}</p>
            <ul className="checklist">
              {appStatus.data.checks.map((check) => (
                <li key={check.name}>{check.message}</li>
              ))}
              <li>
                {appStatus.data.archiveRetentionDays === null
                  ? copy.common.archiveRetentionIndefinite
                  : copy.common.archiveRetentionDays(appStatus.data.archiveRetentionDays)}
              </li>
              <li>{copy.settings.overviewInstalledAt(appStatus.data.runtimeOwnershipPath)}</li>
            </ul>
          </>
        ) : null}
      </section>

      <section className="panel">
        <h2>{copy.common.boardSummary}</h2>
        {snapshot.isPending ? (
          <p className="meta">{copy.common.loading}</p>
        ) : snapshot.error ? (
          <p className="form-error">{snapshot.error.message}</p>
        ) : snapshot.data ? (
          <>
            <div className="stats-grid">
              {boardCounts.map((item) => (
                <div key={item.label} className="stat-card">
                  <span>{item.label}</span>
                  <strong>{item.value}</strong>
                </div>
              ))}
            </div>
            <p className="meta">
              {snapshot.data.session
                ? copy.common.sessionExpiresAt(formatDateTime(snapshot.data.session.expiresAt))
                : copy.common.sessionSignedOut}
            </p>
          </>
        ) : null}
      </section>

      <section className="panel">
        <h2>{copy.settings.archiveTitle}</h2>
        <p className="meta">{copy.settings.archiveHint}</p>
        <form className="board-form" onSubmit={handleSettingsSubmit}>
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={archiveRetentionEnabled}
              onChange={(event) => setArchiveRetentionEnabled(event.target.checked)}
            />
            {copy.settings.archiveAutoDelete}
          </label>
          {archiveRetentionEnabled ? (
            <label className="form-field" htmlFor="archive-retention-days">
              {copy.settings.archiveRetentionLabel}
            </label>
          ) : null}
          {archiveRetentionEnabled ? (
            <input
              id="archive-retention-days"
              className="text-input"
              type="number"
              min="1"
              step="1"
              value={archiveRetentionDays}
              onChange={(event) => setArchiveRetentionDays(event.target.value)}
            />
          ) : null}
          {settingsError ? (
            <p className="form-error" role="alert">
              {settingsError}
            </p>
          ) : null}
          {settingsStatus ? <p className="form-status">{settingsStatus}</p> : null}
          <button className="nav-pill nav-pill-active auth-submit" type="submit" disabled={settingsMutation.isPending}>
            {settingsMutation.isPending ? copy.settings.archiveSaving : copy.settings.archiveSave}
          </button>
        </form>
      </section>

      <section className="panel">
        <h2>{copy.settings.passwordTitle}</h2>
        <p className="meta">{copy.settings.passwordHint}</p>
        <form className="board-form" onSubmit={handlePasswordSubmit}>
          <label className="form-field" htmlFor="current-password">
            {copy.common.currentPassword}
          </label>
          <input
            id="current-password"
            className="text-input"
            type="password"
            autoComplete="current-password"
            value={currentPassword}
            onChange={(event) => setCurrentPassword(event.target.value)}
          />

          <label className="form-field" htmlFor="new-password">
            {copy.common.newPassword}
          </label>
          <input
            id="new-password"
            className="text-input"
            type="password"
            autoComplete="new-password"
            value={newPassword}
            onChange={(event) => setNewPassword(event.target.value)}
          />

          <label className="form-field" htmlFor="confirm-password">
            {copy.settings.passwordConfirmLabel}
          </label>
          <input
            id="confirm-password"
            className="text-input"
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
          />

          {confirmPassword && confirmPassword !== newPassword ? (
            <p className="form-error" role="alert">
              {copy.settings.passwordMismatch}
            </p>
          ) : null}
          {passwordError ? (
            <p className="form-error" role="alert">
              {passwordError}
            </p>
          ) : null}
          {passwordStatus ? <p className="form-status">{passwordStatus}</p> : null}
          <button
            className="nav-pill nav-pill-active auth-submit"
            type="submit"
            disabled={changePasswordMutation.isPending || newPassword !== confirmPassword}
          >
            {changePasswordMutation.isPending ? copy.settings.passwordUpdating : copy.settings.passwordUpdate}
          </button>
        </form>
      </section>

      <section className="panel panel-wide">
        <h2>{copy.settings.sessionsTitle}</h2>
        <p className="meta">{activeSessionsSummary}</p>
        <div className="archive-list">
          {sessions.map((session) => (
            <div key={session.token} className="archive-item">
              <div>
                <strong>{session.current ? copy.common.thisBrowser : copy.common.anotherBrowser}</strong>
                <p className="meta">
                  {copy.settings.sessionMeta(formatDateTime(session.lastSeenAt), formatDateTime(session.expiresAt))}
                </p>
                <p className="meta">{copy.settings.sessionIP(session.clientIP || copy.common.unknown)}</p>
              </div>
              <button
                className="action-button action-button-secondary"
                type="button"
                disabled={revokeSessionMutation.isPending}
                onClick={() => {
                  void revokeSessionMutation.mutateAsync(session.token);
                }}
              >
                {session.current ? copy.settings.signOutHere : copy.settings.revoke}
              </button>
            </div>
          ))}
        </div>
      </section>

      <section className="panel panel-wide">
        <h2>{copy.settings.backupTitle}</h2>
        <p className="meta">{copy.settings.backupHint}</p>
        <div className="board-form">
          <button className="nav-pill nav-pill-active auth-submit" type="button" onClick={() => exportMutation.mutate()} disabled={exportMutation.isPending}>
            {exportMutation.isPending ? copy.settings.exportPreparing : copy.settings.exportButton}
          </button>
          <label className="form-field" htmlFor="import-file">
            {copy.settings.importLabel}
          </label>
          <input id="import-file" className="text-input" type="file" accept="application/json" onChange={handleImportFile} />
          {importError ? (
            <p className="form-error" role="alert">
              {importError}
            </p>
          ) : null}
          {importStatus ? <p className="form-status">{importStatus}</p> : null}
        </div>
      </section>
    </div>
  );
}

function downloadExport(bundle: ExportBundle) {
  const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = `flux-board-export-${new Date(bundle.exportedAt).toISOString().slice(0, 10)}.json`;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
