import { useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useMemo, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";

import { ApiError, bootstrapWithPassword } from "../lib/api";
import { usePreferences } from "../lib/usePreferences";
import { setAuthSessionData, useAuthSession } from "../lib/useAuthSession";
import { bootstrapStatusQueryKey, useBootstrapStatus } from "../lib/useBootstrapStatus";

export function SetupPage() {
  const session = useAuthSession();
  const bootstrap = useBootstrapStatus();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { copy } = usePreferences();
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const readyToBootstrap = useMemo(() => bootstrap.data?.needsSetup ?? false, [bootstrap.data]);

  if (session.isPending || bootstrap.isPending) {
    return (
      <section className="panel" aria-live="polite">
        <h2>{copy.auth.preparingSetupTitle}</h2>
        <p className="meta">{copy.auth.preparingSetupMessage}</p>
      </section>
    );
  }

  if (session.error || bootstrap.error) {
    return (
      <section className="panel panel-error" role="alert">
        <h2>{copy.auth.setupErrorTitle}</h2>
        <p>{session.error?.message ?? bootstrap.error?.message}</p>
      </section>
    );
  }

  if (!readyToBootstrap) {
    return <Navigate replace to={session.data ? "/board" : "/login"} />;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (password !== confirmPassword) {
      setSubmitError(copy.auth.setupPasswordsMustMatch);
      return;
    }

    setSubmitError(null);
    setIsSubmitting(true);

    try {
      const authSession = await bootstrapWithPassword(password);
      setAuthSessionData(queryClient, authSession);
      await queryClient.invalidateQueries({ queryKey: bootstrapStatusQueryKey });
      navigate("/board", { replace: true });
    } catch (error) {
      if (error instanceof ApiError || error instanceof Error) {
        setSubmitError(error.message);
      } else {
        setSubmitError(copy.auth.setupFailed);
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="auth-layout">
      <section className="panel">
        <h2>{copy.auth.setupHeading}</h2>
        <p className="meta">{copy.auth.setupMessage}</p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="form-field" htmlFor="setup-password">
            {copy.common.newPassword}
          </label>
          <input
            id="setup-password"
            className="text-input"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder={copy.auth.setupPasswordPlaceholder}
          />

          <label className="form-field" htmlFor="setup-confirm-password">
            {copy.common.confirmPassword}
          </label>
          <input
            id="setup-confirm-password"
            className="text-input"
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
            placeholder={copy.auth.setupConfirmPlaceholder}
          />

          {submitError ? (
            <p className="form-error" role="alert">
              {submitError}
            </p>
          ) : null}
          <button className="nav-pill nav-pill-active auth-submit" type="submit" disabled={isSubmitting}>
            {isSubmitting ? copy.auth.setupSubmitting : copy.auth.setupSubmit}
          </button>
        </form>
      </section>

      <section className="panel panel-secondary">
        <h2>{copy.auth.whatHappensNext}</h2>
        <ul className="checklist">
          <li>{copy.auth.nextPasswordSaved}</li>
          <li>{copy.auth.nextAutoSignIn}</li>
          <li>{copy.auth.nextSettings}</li>
        </ul>
      </section>
    </div>
  );
}
