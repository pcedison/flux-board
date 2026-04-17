import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";

import { AppShell } from "./AppShell";
import { useAuthSession } from "../lib/useAuthSession";

vi.mock("../lib/useAuthSession", async () => {
  const actual = await vi.importActual<typeof import("../lib/useAuthSession")>("../lib/useAuthSession");
  return {
    ...actual,
    useAuthSession: vi.fn(),
  };
});

const mockedUseAuthSession = vi.mocked(useAuthSession);

describe("AppShell", () => {
  beforeEach(() => {
    mockedUseAuthSession.mockReset();
  });

  it("exposes the shell landmarks and skip link without axe violations", async () => {
    mockedUseAuthSession.mockReturnValue({
      data: null,
      error: null,
      isPending: false,
    } as ReturnType<typeof useAuthSession>);

    const { container } = renderShell();

    expect(screen.getByRole("link", { name: "Skip to main content" })).toHaveAttribute("href", "#main-content");
    expect(screen.getByRole("navigation", { name: "Next UI routes" })).toBeInTheDocument();
    expect(screen.getByRole("main")).toHaveAttribute("id", "main-content");

    const nav = screen.getByRole("navigation", { name: "Next UI routes" });
    expect(within(nav).getByRole("link", { name: "Overview" })).toBeInTheDocument();
    expect(within(nav).getByRole("link", { name: "Sign In" })).toBeInTheDocument();

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
