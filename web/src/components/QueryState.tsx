import type { ReactNode } from "react";

type QueryStateProps = {
  children: ReactNode;
  error: Error | null;
  errorTitle: string;
  isPending: boolean;
  loadingMessage: string;
};

export function QueryState({
  children,
  error,
  errorTitle,
  isPending,
  loadingMessage,
}: QueryStateProps) {
  if (isPending) {
    return (
      <section className="panel" aria-live="polite">
        <h2>Loading</h2>
        <p className="meta">{loadingMessage}</p>
      </section>
    );
  }

  if (error) {
    return (
      <section className="panel panel-error" role="alert">
        <h2>{errorTitle}</h2>
        <p>{error.message}</p>
      </section>
    );
  }

  return <>{children}</>;
}
