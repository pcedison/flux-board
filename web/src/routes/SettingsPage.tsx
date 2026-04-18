import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, type FormEvent, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { QueryState } from "../components/QueryState";
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
import { boardSnapshotQueryKey } from "../lib/useBoardSnapshot";
import { clearAuthSessionData } from "../lib/useAuthSession";

const settingsQueryKey = ["settings"] as const;
const sessionsQueryKey = ["settings-sessions"] as const;

export function SettingsPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const settings = useQuery({ queryKey: settingsQueryKey, queryFn: fetchSettings });
  const sessions = useQuery({ queryKey: sessionsQueryKey, queryFn: fetchSessions });

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
          ? "Archived cards will now stay indefinitely."
          : `Archived cards will auto-delete after ${nextSettings.archiveRetentionDays} days.`,
      );
    },
    onError: (error) => {
      setSettingsStatus(null);
      setSettingsError(error instanceof Error ? error.message : "Unable to save settings.");
    },
  });

  const changePasswordMutation = useMutation({
    mutationFn: ({ currentPassword, newPassword }: { currentPassword: string; newPassword: string }) =>
      changePassword(currentPassword, newPassword),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: sessionsQueryKey });
      setPasswordError(null);
      setPasswordStatus("Password updated. Other sessions were signed out.");
    },
    onError: (error) => {
      setPasswordStatus(null);
      setPasswordError(error instanceof Error ? error.message : "Unable to update password.");
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
      setImportStatus("Backup downloaded.");
    },
    onError: (error) => {
      setImportStatus(null);
      setImportError(error instanceof Error ? error.message : "Unable to download backup.");
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
      setImportStatus("Board restored from backup.");
    },
    onError: (error) => {
      setImportStatus(null);
      setImportError(error instanceof Error ? error.message : "Unable to restore backup.");
    },
  });

  return (
    <QueryState
      error={settings.error ?? sessions.error}
      errorTitle="Unable to load settings"
      isPending={settings.isPending || sessions.isPending}
      loadingMessage="Loading security, retention, and backup controls."
    >
      {settings.data && sessions.data ? (
        <SettingsContent
          archiveRetentionDays={archiveRetentionDays}
          archiveRetentionEnabled={archiveRetentionEnabled}
          changePasswordMutation={changePasswordMutation}
          exportMutation={exportMutation}
          importError={importError}
          importMutation={importMutation}
          importStatus={importStatus}
          passwordError={passwordError}
          passwordStatus={passwordStatus}
          revokeSessionMutation={revokeSessionMutation}
          sessions={sessions.data}
          setArchiveRetentionDays={setDraftArchiveRetentionDays}
          setArchiveRetentionEnabled={setDraftArchiveRetentionEnabled}
          settingsError={settingsError}
          settingsMutation={updateSettingsMutation}
          settingsStatus={settingsStatus}
        />
      ) : null}
    </QueryState>
  );
}

function SettingsContent({
  archiveRetentionDays,
  archiveRetentionEnabled,
  changePasswordMutation,
  exportMutation,
  importError,
  importMutation,
  importStatus,
  passwordError,
  passwordStatus,
  revokeSessionMutation,
  sessions,
  setArchiveRetentionDays,
  setArchiveRetentionEnabled,
  settingsError,
  settingsMutation,
  settingsStatus,
}: {
  archiveRetentionDays: string;
  archiveRetentionEnabled: boolean;
  changePasswordMutation: ReturnType<typeof useMutation<void, Error, { currentPassword: string; newPassword: string }>>;
  exportMutation: ReturnType<typeof useMutation<ExportBundle, Error, void>>;
  importError: string | null;
  importMutation: ReturnType<typeof useMutation<void, Error, ExportBundle>>;
  importStatus: string | null;
  passwordError: string | null;
  passwordStatus: string | null;
  revokeSessionMutation: ReturnType<typeof useMutation<void, Error, string>>;
  sessions: Awaited<ReturnType<typeof fetchSessions>>;
  setArchiveRetentionDays: (value: string) => void;
  setArchiveRetentionEnabled: (value: boolean) => void;
  settingsError: string | null;
  settingsMutation: ReturnType<typeof useMutation<AppSettings, Error, AppSettings>>;
  settingsStatus: string | null;
}) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const activeSessionsSummary = useMemo(
    () => `${sessions.length} active ${sessions.length === 1 ? "session" : "sessions"}`,
    [sessions.length],
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

  return (
    <div className="page-grid settings-grid">
      <section className="panel">
        <h2>Archive policy</h2>
        <p className="meta">Choose whether archived cards stay forever or expire automatically.</p>
        <form className="board-form" onSubmit={handleSettingsSubmit}>
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={archiveRetentionEnabled}
              onChange={(event) => setArchiveRetentionEnabled(event.target.checked)}
            />
            Auto-delete archived cards
          </label>
          {archiveRetentionEnabled ? (
            <label className="form-field" htmlFor="archive-retention-days">
              Retention (days)
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
            {settingsMutation.isPending ? "Saving..." : "Save archive policy"}
          </button>
        </form>
      </section>

      <section className="panel">
        <h2>Password</h2>
        <p className="meta">Change the board password without running setup again.</p>
        <form className="board-form" onSubmit={handlePasswordSubmit}>
          <label className="form-field" htmlFor="current-password">
            Current password
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
            New password
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
            Confirm new password
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
              New passwords must match.
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
            {changePasswordMutation.isPending ? "Updating..." : "Update password"}
          </button>
        </form>
      </section>

      <section className="panel">
        <h2>Sessions</h2>
        <p className="meta">{activeSessionsSummary}</p>
        <div className="archive-list">
          {sessions.map((session) => (
            <div key={session.token} className="archive-item">
              <div>
                <strong>{session.current ? "This browser" : "Another signed-in browser"}</strong>
                <p className="meta">
                  Last active {new Date(session.lastSeenAt).toLocaleString()} • expires{" "}
                  {new Date(session.expiresAt).toLocaleString()}
                </p>
                <p className="meta">IP address {session.clientIP || "unknown"}</p>
              </div>
              <button
                className="action-button action-button-secondary"
                type="button"
                disabled={revokeSessionMutation.isPending}
                onClick={() => {
                  void revokeSessionMutation.mutateAsync(session.token);
                }}
              >
                {session.current ? "Sign out here" : "Revoke"}
              </button>
            </div>
          ))}
        </div>
      </section>

      <section className="panel">
        <h2>Backup & restore</h2>
        <p className="meta">Download a full backup or restore the board from an earlier export.</p>
        <div className="board-form">
          <button className="nav-pill nav-pill-active auth-submit" type="button" onClick={() => exportMutation.mutate()} disabled={exportMutation.isPending}>
            {exportMutation.isPending ? "Preparing backup..." : "Download backup"}
          </button>
          <label className="form-field" htmlFor="import-file">
            Restore from export
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
