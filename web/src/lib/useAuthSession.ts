import { useQuery } from "@tanstack/react-query";

import { fetchAuthSession } from "./api";

export function useAuthSession() {
  return useQuery({
    queryKey: ["auth-session"],
    queryFn: fetchAuthSession,
  });
}
