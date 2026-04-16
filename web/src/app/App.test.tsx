import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { App } from "./App";
import type { BoardSnapshot, AuthSession } from "../lib/api";
import { useAuthSession } from "../lib/useAuthSession";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

vi.mock("../lib/useAuthSession", () => ({
  useAuthSession: vi.fn(),
}));

vi.mock("../lib/useBoardSnapshot", () => ({
  useBoardSnapshot: vi.fn(),
}));

const mockedUseAuthSession = vi.mocked(useAuthSession);
const mockedUseBoardSnapshot = vi.mocked(useBoardSnapshot);

function renderApp(initialEntry: string) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function mockAuthSession(data: AuthSession | null, isPending = false) {
  mockedUseAuthSession.mockReturnValue(
    {
      data,
      error: null,
      isPending,
    } as unknown as ReturnType<typeof useAuthSession>,
  );
}

function mockBoardSnapshot(data: BoardSnapshot | undefined, isPending = false) {
  mockedUseBoardSnapshot.mockReturnValue(
    {
      data,
      error: null,
      isPending,
    } as unknown as ReturnType<typeof useBoardSnapshot>,
  );
}

describe("App auth-aware routing", () => {
  beforeEach(() => {
    mockedUseAuthSession.mockReset();
    mockedUseBoardSnapshot.mockReset();
  });

  it("redirects unauthenticated board visits to the login route", () => {
    mockAuthSession(null);
    mockBoardSnapshot(undefined, true);

    renderApp("/board");

    expect(screen.getByRole("heading", { name: "Sign in to view the board" })).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByText(/Redirects you back to/i)).toHaveTextContent("/board");
  });

  it("allows authenticated visitors through to the board snapshot route", () => {
    mockAuthSession({ authenticated: true, expiresAt: 1 });
    mockBoardSnapshot({
        session: { authenticated: true, expiresAt: 1 },
        tasks: [
          {
            id: "task-1",
            title: "Queue me",
            note: "",
            due: "2026-04-20",
            priority: "medium",
            status: "queued",
            sort_order: 0,
            lastUpdated: 1,
          },
        ],
        archived: [],
      });

    renderApp("/board");

    expect(screen.getByText("Queue me")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Sign in to view the board" })).not.toBeInTheDocument();
  });

  it("redirects authenticated login visits back to the protected board route", () => {
    mockAuthSession({ authenticated: true, expiresAt: 1 });
    mockBoardSnapshot({
        session: { authenticated: true, expiresAt: 1 },
        tasks: [],
        archived: [],
      });

    renderApp("/login");

    expect(screen.getByText("Archive Snapshot")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Sign in to view the board" })).not.toBeInTheDocument();
  });
});
