import type { PropsWithChildren } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { NavLink } from "react-router-dom";
import { useLocation, useNavigate } from "react-router-dom";

import { logout } from "../lib/api";
import { usePreferences } from "../lib/usePreferences";
import { boardSnapshotQueryKey } from "../lib/useBoardSnapshot";
import { clearAuthSessionData, useAuthSession } from "../lib/useAuthSession";
import { useBootstrapStatus } from "../lib/useBootstrapStatus";

export function AppShell({ children }: PropsWithChildren) {
  const session = useAuthSession();
  const bootstrap = useBootstrapStatus();
  const { copy, isDarkTheme, toggleTheme } = usePreferences();
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

  const navItems = session.data
    ? [
        { href: "/board", label: copy.common.board },
        { href: "/settings", label: copy.common.settings },
      ]
    : [];

  return (
    <>
      <a className="skip-link" href="#main-content">
        {copy.shell.skipLink}
      </a>
      <div className="app-shell">
        <header className="hero">
          <div className="hero-brand">
            <div className="hero-brand-mark" aria-hidden="true">
              <svg width="12" height="12" viewBox="0 0 14 14" fill="white">
                <path d="M2 2h4v4H2zM8 2h4v4H8zM2 8h4v4H2zM8 8h4v4H8z"/>
              </svg>
            </div>
            <h1>{copy.common.appName}</h1>
          </div>

          <nav className="hero-nav" aria-label={copy.shell.navLabel}>
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
          </nav>

          <div className="hero-actions">
            <button
              className="nav-pill nav-pill-muted nav-button theme-toggle"
              type="button"
              aria-label={isDarkTheme ? copy.shell.themeToggleToLight : copy.shell.themeToggleToDark}
              onClick={toggleTheme}
            >
              {isDarkTheme ? copy.common.light : copy.common.dark}
            </button>
            {session.isPending || bootstrap.isPending ? (
              <span className="nav-pill nav-pill-muted" aria-live="polite">
                {copy.common.checkingAccess}
              </span>
            ) : session.data ? (
              <button
                className="nav-pill nav-pill-muted nav-button"
                type="button"
                onClick={() => logoutMutation.mutate()}
                disabled={logoutMutation.isPending}
              >
                {logoutMutation.isPending ? copy.common.signingOut : copy.common.signOut}
              </button>
            ) : bootstrap.data?.needsSetup ? (
              <NavLink
                to="/setup"
                className={({ isActive }) => (isActive ? "nav-pill nav-pill-active" : "nav-pill")}
              >
                {copy.common.setup}
              </NavLink>
            ) : (
              <NavLink
                to="/login"
                className={({ isActive }) => (isActive ? "nav-pill nav-pill-active" : "nav-pill")}
              >
                {copy.common.signIn}
              </NavLink>
            )}
          </div>
        </header>

        <main id="main-content" className="page-shell">
          {children}
        </main>
      </div>
    </>
  );
}
