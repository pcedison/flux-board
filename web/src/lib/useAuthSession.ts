import type { QueryClient } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";

import type { AuthSession } from "./api";
import { fetchAuthSession } from "./api";

export const authSessionQueryKey = ["auth-session"] as const;

export function setAuthSessionData(queryClient: QueryClient, session: AuthSession | null) {
  queryClient.setQueryData(authSessionQueryKey, session);
}

export function clearAuthSessionData(queryClient: QueryClient) {
  setAuthSessionData(queryClient, null);
}

export function useAuthSession() {
  return useQuery({
    queryKey: authSessionQueryKey,
    queryFn: fetchAuthSession,
  });
}
