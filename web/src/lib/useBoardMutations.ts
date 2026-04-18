import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  archiveTask,
  createTask,
  deleteArchivedTask,
  isUnauthorizedApiError,
  moveTask,
  restoreTask,
  updateTask,
  type MoveTaskInput,
  type TaskDraft,
  type TaskUpdateDraft,
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
    updateTask: useMutation({
      mutationFn: ({ id, task }: { id: string; task: TaskUpdateDraft }) => updateTask(id, task),
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
    deleteArchivedTask: useMutation({
      mutationFn: (id: string) => deleteArchivedTask(id),
      onSuccess: invalidateBoardSnapshot,
      onError: handleMutationError,
    }),
  };
}
