import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, expect } from "vitest";

const { toHaveNoViolations } = (await import("vitest-axe/matchers")) as {
  toHaveNoViolations: Parameters<typeof expect.extend>[0]["toHaveNoViolations"];
};

expect.extend({ toHaveNoViolations });

afterEach(() => {
  cleanup();
});
