export function resolveRouterBasename(pathname: string): string {
  if (pathname === "/next" || pathname.startsWith("/next/")) {
    return "/next";
  }

  return "";
}

export function currentRouterBasename() {
  if (typeof window === "undefined") {
    return "";
  }

  return resolveRouterBasename(window.location.pathname);
}
