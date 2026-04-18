import type { PropsWithChildren } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { NavLink } from "react-router-dom";
import { useLocation, useNavigate } from "react-router-dom";

import { logout } from "../lib/api";
import { boardSnapshotQueryKey } from "../lib/useBoardSnapshot";
import { clearAuthSessionData, useAuthSession } from "../lib/useAuthSession";
import { useBootstrapStatus } from "../lib/useBootstrapStatus";

export function AppShell({ children }: PropsWithChildren) {
  const session = useAuthSession();
  const bootstrap = useBootstrapStatus();
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
    { href: "/status", label: "Status" },
    ...(session.data ? [{ href: "/board", label: "Board" }, { href: "/settings", label: "Settings" }] : []),
  ];

  return (
    <>
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
      <div className="app-shell">
        <header className="hero">
          <div className="hero-copy">
            <p className="eyebrow">Self-Hosted Personal Board</p>
            <h1>Flux Board</h1>
            <p className="lede">
              A focused planning board for one person. Flux Board keeps tasks, archived work,
              backups, and sign-in controls together in one simple app you can run yourself.
            </p>
          </div>
          <nav className="hero-nav" aria-label="Primary routes">
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
            {session.isPending || bootstrap.isPending ? (
              <span className="nav-pill nav-pill-muted" aria-live="polite">
                Checking access
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
            ) : bootstrap.data?.needsSetup ? (
              <NavLink
                to="/setup"
                className={({ isActive }) => (isActive ? "nav-pill nav-pill-active" : "nav-pill")}
              >
                Setup
              </NavLink>
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
