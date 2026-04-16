import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  archiveTask,
  createTask,
  isUnauthorizedApiError,
  moveTask,
  restoreTask,
  type MoveTaskInput,
  type TaskDraft,
} from "./api";
import { boardSnapshotQueryKey } from "./useBoardSnapshot";
import { clearAuthSessionData } from "./useAuthSession";

export function useBoardMutations() {
  const queryClient = useQueryClient();

  const invalidateBoardSnapshot = async () => {
    await queryClient.invalidateQueries({ queryKey: boardSnapshotQueryKey });
  };

  const handleMutationError = (error: unknown) => {
    if (isUnauthorizedApiError(error)) {
      clearAuthSessionData(queryClient);
      void queryClient.invalidateQueries({ queryKey: boardSnapshotQueryKey });
    }
  };

  return {
    createTask: useMutation({
      mutationFn: (task: TaskDraft) => createTask(task),
      onSuccess: invalidateBoardSnapshot,
      onError: handleMutationError,
    }),
    moveTask: useMutation({
      mutationFn: (input: MoveTaskInput) => moveTask(input),
      onSuccess: invalidateBoardSnapshot,
      onError: handleMutationError,
    }),
    archiveTask: useMutation({
      mutationFn: (id: string) => archiveTask(id),
      onSuccess: invalidateBoardSnapshot,
      onError: handleMutationError,
    }),
    restoreTask: useMutation({
      mutationFn: (id: string) => restoreTask(id),
      onSuccess: invalidateBoardSnapshot,
      onError: handleMutationError,
    }),
  };
}
