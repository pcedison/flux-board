import { Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom";

import { AppShell } from "../components/AppShell";
import { QueryState } from "../components/QueryState";
import { usePreferences } from "../lib/usePreferences";
import { useAuthSession } from "../lib/useAuthSession";
import { useBootstrapStatus } from "../lib/useBootstrapStatus";
import { BoardSnapshotPage } from "../routes/BoardSnapshotPage";
import { LoginPage } from "../routes/LoginPage";
import { SettingsPage } from "../routes/SettingsPage";
import { SetupPage } from "../routes/SetupPage";

export function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/status" element={<Navigate replace to="/settings" />} />
        <Route path="/about" element={<Navigate replace to="/settings" />} />
        <Route path="/setup" element={<SetupPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route element={<RequireAuthRoute />}>
          <Route path="/board" element={<BoardSnapshotPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
        <Route path="*" element={<Navigate replace to="/" />} />
      </Routes>
    </AppShell>
  );
}

function HomePage() {
  const bootstrap = useBootstrapStatus();
  const session = useAuthSession();
  const { copy } = usePreferences();

  return (
    <QueryState
      error={bootstrap.error ?? session.error}
      errorTitle={copy.auth.unableToOpenApp(copy.common.appName)}
      isPending={bootstrap.isPending || session.isPending}
      loadingMessage={copy.auth.decidingRoute}
    >
      {bootstrap.data ? (
        bootstrap.data.needsSetup ? (
          <Navigate replace to="/setup" />
        ) : session.data ? (
          <Navigate replace to="/board" />
        ) : (
          <Navigate replace to="/login" />
        )
      ) : null}
    </QueryState>
  );
}

function RequireAuthRoute() {
  const location = useLocation();
  const session = useAuthSession();
  const { copy } = usePreferences();

  return (
    <QueryState
      error={session.error}
      errorTitle={copy.auth.verifySessionTitle}
      isPending={session.isPending}
      loadingMessage={copy.auth.verifySessionMessage}
    >
      {session.data ? (
        <Outlet />
      ) : (
        <Navigate
          replace
          to="/login"
          state={{ from: `${location.pathname}${location.search}${location.hash}` }}
        />
      )}
    </QueryState>
  );
}
