import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { LoginPage } from "./LoginPage";
import { useAuthSession } from "../lib/useAuthSession";
import { useBootstrapStatus } from "../lib/useBootstrapStatus";

vi.mock("../lib/useAuthSession", async () => {
  const actual = await vi.importActual<typeof import("../lib/useAuthSession")>("../lib/useAuthSession");
  return {
    ...actual,
    useAuthSession: vi.fn(),
  };
});

vi.mock("../lib/useBootstrapStatus", async () => {
  const actual = await vi.importActual<typeof import("../lib/useBootstrapStatus")>("../lib/useBootstrapStatus");
  return {
    ...actual,
    useBootstrapStatus: vi.fn(),
  };
});

const mockedUseAuthSession = vi.mocked(useAuthSession);
const mockedUseBootstrapStatus = vi.mocked(useBootstrapStatus);

describe("LoginPage", () => {
  beforeEach(() => {
    mockedUseAuthSession.mockReset();
    mockedUseBootstrapStatus.mockReset();
  });

  it("keeps the sign-in form accessible and free of axe violations", async () => {
    mockedUseAuthSession.mockReturnValue({
      data: null,
      error: null,
      isPending: false,
    } as ReturnType<typeof useAuthSession>);
    mockedUseBootstrapStatus.mockReturnValue({
      data: { needsSetup: false },
      error: null,
      isPending: false,
    } as ReturnType<typeof useBootstrapStatus>);

    const { container } = renderLoginPage();

    expect(screen.getByRole("heading", { name: "Sign in to view the board" })).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
    expect(screen.getByRole("button", { name: "Sign In" })).toBeEnabled();
    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });

  it("renders the session error panel with accessible alert semantics", async () => {
    mockedUseAuthSession.mockReturnValue({
      data: null,
      error: new Error("session lookup failed"),
      isPending: false,
    } as ReturnType<typeof useAuthSession>);
    mockedUseBootstrapStatus.mockReturnValue({
      data: { needsSetup: false },
      error: null,
      isPending: false,
    } as ReturnType<typeof useBootstrapStatus>);

    const { container } = renderLoginPage();

    expect(screen.getByRole("alert")).toHaveTextContent("Unable to open the sign-in route");
    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });
});

function renderLoginPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}
