import { useQuery, useQueryClient } from "@tanstack/react-query";

import { fetchBoardSnapshot } from "./api";
import { clearAuthSessionData } from "./useAuthSession";

export const boardSnapshotQueryKey = ["board-snapshot"] as const;

export function useBoardSnapshot() {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: boardSnapshotQueryKey,
    queryFn: async () => {
      const snapshot = await fetchBoardSnapshot();
      if (!snapshot.session) {
        clearAuthSessionData(queryClient);
      }
      return snapshot;
    },
  });
}
