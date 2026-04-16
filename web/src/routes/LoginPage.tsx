import { useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useMemo, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";

import { ApiError, loginWithPassword } from "../lib/api";
import { setAuthSessionData, useAuthSession } from "../lib/useAuthSession";

type LoginLocationState = {
  from?: string;
};

export function LoginPage() {
  const session = useAuthSession();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const [password, setPassword] = useState("");
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const nextPath = useMemo(() => {
    const state = location.state as LoginLocationState | null;
    return state?.from?.startsWith("/") ? state.from : "/board";
  }, [location.state]);

  if (session.isPending) {
    return (
      <section className="panel" aria-live="polite">
        <h2>Checking session</h2>
        <p className="meta">Confirming whether this browser already has an active Flux Board session.</p>
      </section>
    );
  }

  if (session.error) {
    return (
      <section className="panel panel-error" role="alert">
        <h2>Unable to open the sign-in route</h2>
        <p>{session.error.message}</p>
      </section>
    );
  }

  if (session.data) {
    return <Navigate replace to={nextPath} />;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!password.trim()) {
      setSubmitError("Enter the current Flux Board password to continue.");
      return;
    }

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      const authSession = await loginWithPassword(password);
      setAuthSessionData(queryClient, authSession);
      await queryClient.invalidateQueries({ queryKey: ["board-snapshot"] });
      navigate(nextPath, { replace: true });
    } catch (error) {
      if (error instanceof ApiError) {
        setSubmitError(error.message);
      } else if (error instanceof Error) {
        setSubmitError(error.message);
      } else {
        setSubmitError("Sign-in failed.");
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="auth-layout">
      <section className="panel">
        <h2>Sign in to view the board</h2>
        <p className="meta">
          This W7 slice keeps board data read-only, but it now respects the real session boundary. Use
          the existing Flux Board password here to open the protected board snapshot route.
        </p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="form-field" htmlFor="login-password">
            Password
          </label>
          <input
            id="login-password"
            className="text-input"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => {
              setPassword(event.target.value);
              if (submitError) {
                setSubmitError(null);
              }
            }}
            placeholder="Enter the current Flux Board password"
          />
          {submitError ? (
            <p className="form-error" role="alert">
              {submitError}
            </p>
          ) : null}
          <button className="nav-pill nav-pill-active auth-submit" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Signing in..." : "Sign in"}
          </button>
        </form>
      </section>

      <section className="panel panel-secondary">
        <h2>What this route does</h2>
        <ul className="checklist">
          <li>Protects `/board` until an authenticated session cookie is present.</li>
          <li>Reuses the current Go auth/session API without introducing board mutations yet.</li>
          <li>
            Redirects you back to <code>{nextPath}</code> once the session is established.
          </li>
        </ul>
      </section>
    </div>
  );
}
