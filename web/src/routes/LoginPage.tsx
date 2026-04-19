import { useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useMemo, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";

import { ApiError, isSetupRequiredApiError, loginWithPassword } from "../lib/api";
import { usePreferences } from "../lib/preferences";
import { setAuthSessionData, useAuthSession } from "../lib/useAuthSession";
import { useBootstrapStatus } from "../lib/useBootstrapStatus";

type LoginLocationState = {
  from?: string;
};

export function LoginPage() {
  const session = useAuthSession();
  const bootstrap = useBootstrapStatus();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const { copy } = usePreferences();
  const [password, setPassword] = useState("");
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const nextPath = useMemo(() => {
    const state = location.state as LoginLocationState | null;
    return state?.from?.startsWith("/") ? state.from : "/board";
  }, [location.state]);

  if (session.isPending || bootstrap.isPending) {
    return (
      <section className="panel" aria-live="polite">
        <h2>{copy.auth.checkingSignInTitle}</h2>
        <p className="meta">{copy.auth.checkingSignInMessage}</p>
      </section>
    );
  }

  if (session.error || bootstrap.error) {
    return (
      <section className="panel panel-error" role="alert">
        <h2>{copy.auth.signInErrorTitle}</h2>
        <p>{session.error?.message ?? bootstrap.error?.message}</p>
      </section>
    );
  }

  if (session.data) {
    return <Navigate replace to={nextPath} />;
  }

  if (bootstrap.data?.needsSetup) {
    return <Navigate replace to="/setup" />;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!password.trim()) {
      setSubmitError(copy.auth.signInRequired);
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
      if (isSetupRequiredApiError(error)) {
        navigate("/setup", { replace: true });
        return;
      }
      if (error instanceof ApiError) {
        setSubmitError(error.message);
      } else if (error instanceof Error) {
        setSubmitError(error.message);
      } else {
        setSubmitError(copy.auth.signInFailed);
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="auth-layout">
      <section className="panel">
        <h2>{copy.auth.signInHeading}</h2>
        <p className="meta">{copy.auth.signInMessage}</p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="form-field" htmlFor="login-password">
            {copy.common.password}
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
            placeholder={copy.auth.signInPlaceholder}
          />
          {submitError ? (
            <p className="form-error" role="alert">
              {submitError}
            </p>
          ) : null}
          <button className="nav-pill nav-pill-active auth-submit" type="submit" disabled={isSubmitting}>
            {isSubmitting ? copy.auth.signInSubmitting : copy.common.signIn}
          </button>
        </form>
      </section>

      <section className="panel panel-secondary">
        <h2>{copy.auth.afterSignIn}</h2>
        <ul className="checklist">
          <li>{copy.auth.afterSignInReturnTo(nextPath)}</li>
          <li>{copy.auth.afterSignInSettings}</li>
          <li>{copy.auth.afterSignInSetup}</li>
        </ul>
      </section>
    </div>
  );
}
