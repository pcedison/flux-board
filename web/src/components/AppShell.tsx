import type { PropsWithChildren } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { NavLink } from "react-router-dom";
import { useLocation, useNavigate } from "react-router-dom";

import { logout } from "../lib/api";
import { boardSnapshotQueryKey } from "../lib/useBoardSnapshot";
import { clearAuthSessionData, useAuthSession } from "../lib/useAuthSession";

export function AppShell({ children }: PropsWithChildren) {
  const session = useAuthSession();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSettled: async () => {
      clearAuthSessionData(queryClient);
      await queryClient.invalidateQueries({ queryKey: boardSnapshotQueryKey });
      navigate("/login", {
        replace: true,
        state: { from: location.pathname === "/login" ? "/board" : location.pathname },
      });
    },
  });

  const navItems = [
    { href: "/", label: "Overview" },
    ...(session.data ? [{ href: "/board", label: "Board Snapshot" }] : []),
  ];

  return (
    <>
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
      <div className="app-shell">
        <header className="hero">
          <div className="hero-copy">
            <p className="eyebrow">W7 Frontend Foundation</p>
            <h1>Flux Board Next UI</h1>
            <p className="lede">
              A read-only React + TypeScript + Vite shell that talks to the real Go API while the
              current embedded frontend stays in service. This keeps the migration path incremental
              and testable.
            </p>
          </div>
          <nav className="hero-nav" aria-label="Next UI routes">
            {navItems.map((item) => (
              <NavLink
                key={item.href}
                to={item.href}
                className={({ isActive }) => (isActive ? "nav-pill nav-pill-active" : "nav-pill")}
                end={item.href === "/"}
              >
                {item.label}
              </NavLink>
            ))}
            {session.isPending ? (
              <span className="nav-pill nav-pill-muted" aria-live="polite">
                Checking session
              </span>
            ) : session.data ? (
              <button
                className="nav-pill nav-pill-muted nav-button"
                type="button"
                onClick={() => logoutMutation.mutate()}
                disabled={logoutMutation.isPending}
              >
                {logoutMutation.isPending ? "Signing out..." : "Sign out"}
              </button>
            ) : (
              <NavLink
                to="/login"
                className={({ isActive }) => (isActive ? "nav-pill nav-pill-active" : "nav-pill")}
              >
                Sign In
              </NavLink>
            )}
          </nav>
        </header>

        <main id="main-content" className="page-shell">
          {children}
        </main>
      </div>
    </>
  );
}
