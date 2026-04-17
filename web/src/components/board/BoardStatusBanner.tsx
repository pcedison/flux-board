type BoardStatusBannerProps = {
  error: string | null;
  status: string | null;
};

export function BoardStatusBanner({ error, status }: BoardStatusBannerProps) {
  if (error) {
    return (
      <section className="panel panel-error board-feedback" role="alert">
        <strong>Board action failed.</strong>
        <p>{error}</p>
      </section>
    );
  }

  if (!status) {
    return null;
  }

  return (
    <section className="panel panel-status board-feedback" role="status">
      <strong>Board updated.</strong>
      <p>{status}</p>
    </section>
  );
}
