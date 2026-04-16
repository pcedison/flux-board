import { useQuery } from "@tanstack/react-query";

import { fetchBoardSnapshot } from "./api";

export function useBoardSnapshot() {
  return useQuery({
    queryKey: ["board-snapshot"],
    queryFn: fetchBoardSnapshot,
  });
}
