import { useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useMemo, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";

import { ApiError, bootstrapWithPassword } from "../lib/api";
import { setAuthSessionData, useAuthSession } from "../lib/useAuthSession";
import { bootstrapStatusQueryKey, useBootstrapStatus } from "../lib/useBootstrapStatus";

export function SetupPage() {
  const session = useAuthSession();
  const bootstrap = useBootstrapStatus();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const readyToBootstrap = useMemo(() => bootstrap.data?.needsSetup ?? false, [bootstrap.data]);

  if (session.isPending || bootstrap.isPending) {
    return (
      <section className="panel" aria-live="polite">
        <h2>Preparing setup</h2>
        <p className="meta">Checking whether this Flux Board instance already has an admin password.</p>
      </section>
    );
  }

  if (session.error || bootstrap.error) {
    return (
      <section className="panel panel-error" role="alert">
        <h2>Unable to open setup</h2>
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
      setSubmitError("Passwords must match before setup can continue.");
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
        setSubmitError("Setup failed.");
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="auth-layout">
      <section className="panel">
        <h2>Set the admin password</h2>
        <p className="meta">
          This is a single-user installation. Create the first password once, then use it for daily sign-in.
        </p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="form-field" htmlFor="setup-password">
            New password
          </label>
          <input
            id="setup-password"
            className="text-input"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="Create a strong admin password"
          />

          <label className="form-field" htmlFor="setup-confirm-password">
            Confirm password
          </label>
          <input
            id="setup-confirm-password"
            className="text-input"
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
            placeholder="Type the password again"
          />

          {submitError ? (
            <p className="form-error" role="alert">
              {submitError}
            </p>
          ) : null}
          <button className="nav-pill nav-pill-active auth-submit" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Finishing setup..." : "Finish setup"}
          </button>
        </form>
      </section>

      <section className="panel panel-secondary">
        <h2>What happens next</h2>
        <ul className="checklist">
          <li>The password is stored as a bcrypt hash in PostgreSQL.</li>
          <li>Flux Board signs this browser in immediately after setup succeeds.</li>
          <li>You can later rotate the password and archive policy from Settings.</li>
        </ul>
      </section>
    </div>
  );
}
