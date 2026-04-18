import { useQuery } from "@tanstack/react-query";

import { fetchAppStatus } from "./api";

export const appStatusQueryKey = ["app-status"] as const;

export function useAppStatus() {
  return useQuery({
    queryKey: appStatusQueryKey,
    queryFn: fetchAppStatus,
  });
}
