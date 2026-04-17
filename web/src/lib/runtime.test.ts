import { describe, expect, it } from "vitest";

import { resolveRouterBasename } from "./runtime";

describe("resolveRouterBasename", () => {
  it("uses /next for preview-runtime paths", () => {
    expect(resolveRouterBasename("/next")).toBe("/next");
    expect(resolveRouterBasename("/next/")).toBe("/next");
    expect(resolveRouterBasename("/next/board")).toBe("/next");
    expect(resolveRouterBasename("/next/login")).toBe("/next");
  });

  it("uses the root basename for non-preview paths", () => {
    expect(resolveRouterBasename("/")).toBe("");
    expect(resolveRouterBasename("/board")).toBe("");
    expect(resolveRouterBasename("/legacy")).toBe("");
  });
});
