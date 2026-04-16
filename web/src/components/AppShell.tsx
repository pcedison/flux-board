import type { PropsWithChildren } from "react";
import { NavLink } from "react-router-dom";

const navItems = [
  { href: "/", label: "Overview" },
  { href: "/board", label: "Board Snapshot" },
  { href: "/login", label: "Sign In" },
];

export function AppShell({ children }: PropsWithChildren) {
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
          </nav>
        </header>

        <main id="main-content" className="page-shell">
          {children}
        </main>
      </div>
    </>
  );
}
