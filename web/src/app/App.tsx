import { Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom";

import { AppShell } from "../components/AppShell";
import { QueryState } from "../components/QueryState";
import { useAuthSession } from "../lib/useAuthSession";
import { BoardSnapshotPage } from "../routes/BoardSnapshotPage";
import { LoginPage } from "../routes/LoginPage";
import { OverviewPage } from "../routes/OverviewPage";

export function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<OverviewPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route element={<RequireAuthRoute />}>
          <Route path="/board" element={<BoardSnapshotPage />} />
        </Route>
        <Route path="*" element={<Navigate replace to="/" />} />
      </Routes>
    </AppShell>
  );
}

function RequireAuthRoute() {
  const location = useLocation();
  const session = useAuthSession();

  return (
    <QueryState
      error={session.error}
      errorTitle="Unable to verify your session"
      isPending={session.isPending}
      loadingMessage="Checking whether the new frontend can safely open the protected board snapshot."
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
