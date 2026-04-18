import { useQuery } from "@tanstack/react-query";

import { fetchBootstrapStatus } from "./api";

export const bootstrapStatusQueryKey = ["bootstrap-status"] as const;

export function useBootstrapStatus() {
  return useQuery({
    queryKey: bootstrapStatusQueryKey,
    queryFn: fetchBootstrapStatus,
  });
}
