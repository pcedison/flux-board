import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { AppShell } from "./AppShell";
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

describe("AppShell", () => {
  beforeEach(() => {
    mockedUseAuthSession.mockReset();
    mockedUseBootstrapStatus.mockReset();
  });

  it("exposes the shell landmarks and skip link without axe violations", async () => {
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

    const { container } = renderShell();

    expect(screen.getByRole("link", { name: "Skip to main content" })).toHaveAttribute("href", "#main-content");
    expect(screen.getByRole("navigation", { name: "Primary routes" })).toBeInTheDocument();
    expect(screen.getByRole("main")).toHaveAttribute("id", "main-content");

    expect(screen.getByRole("navigation", { name: "Primary routes" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign In" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Switch to dark mode" })).toBeInTheDocument();

    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });
});

function renderShell() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/"]}>
        <AppShell>
          <div>Shell content</div>
        </AppShell>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}
