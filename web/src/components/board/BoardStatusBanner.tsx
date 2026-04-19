import { usePreferences } from "../../lib/preferences";

type BoardStatusBannerProps = {
  error: string | null;
  status: string | null;
};

export function BoardStatusBanner({ error, status }: BoardStatusBannerProps) {
  const { copy } = usePreferences();

  if (error) {
    return (
      <section className="panel panel-error board-feedback" role="alert">
        <strong>{copy.board.actionFailedTitle}</strong>
        <p>{error}</p>
      </section>
    );
  }

  if (!status) {
    return null;
  }

  return (
    <section className="panel panel-status board-feedback" role="status">
      <strong>{copy.board.actionUpdatedTitle}</strong>
      <p>{status}</p>
    </section>
  );
}
