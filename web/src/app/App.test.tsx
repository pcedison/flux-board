import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { App } from "./App";
import { logout } from "../lib/api";
import type { BoardSnapshot, AuthSession } from "../lib/api";
import { useAuthSession } from "../lib/useAuthSession";
import { useBoardSnapshot } from "../lib/useBoardSnapshot";

vi.mock("../lib/useAuthSession", async () => {
  const actual = await vi.importActual("../lib/useAuthSession");
  return {
    ...actual,
    useAuthSession: vi.fn(),
  };
});

vi.mock("../lib/useBoardSnapshot", async () => {
  const actual = await vi.importActual("../lib/useBoardSnapshot");
  return {
    ...actual,
    useBoardSnapshot: vi.fn(),
  };
});

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    logout: vi.fn().mockResolvedValue(undefined),
  };
});

const mockedUseAuthSession = vi.mocked(useAuthSession);
const mockedUseBoardSnapshot = vi.mocked(useBoardSnapshot);
const mockedLogout = vi.mocked(logout);

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
    mockedLogout.mockClear().mockResolvedValue(undefined);
  });

  it("redirects unauthenticated board visits to the login route", () => {
    mockAuthSession(null);
    mockBoardSnapshot(undefined, true);

    renderApp("/board");

    expect(screen.getByRole("heading", { name: "Sign in to view the board" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign In" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Board" })).not.toBeInTheDocument();
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
    expect(screen.getByRole("link", { name: "Board" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign out" })).toBeInTheDocument();
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

  it("clears shell auth ownership after logout", async () => {
    mockedUseAuthSession.mockReturnValueOnce(
      {
        data: { authenticated: true, expiresAt: 1 },
        error: null,
        isPending: false,
      } as unknown as ReturnType<typeof useAuthSession>,
    );
    mockedUseAuthSession.mockReturnValue(
      {
        data: null,
        error: null,
        isPending: false,
      } as unknown as ReturnType<typeof useAuthSession>,
    );
    mockBoardSnapshot(undefined, true);

    renderApp("/");

    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));

    await waitFor(() => expect(mockedLogout).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByRole("link", { name: "Sign In" })).toBeInTheDocument());
  });
});
