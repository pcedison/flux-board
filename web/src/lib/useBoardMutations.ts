import { useMutation, useQueryClient } from "@tanstack/react-query";

import { archiveTask, createTask, moveTask, restoreTask, type MoveTaskInput, type TaskDraft } from "./api";

const boardSnapshotQueryKey = ["board-snapshot"] as const;

export function useBoardMutations() {
  const queryClient = useQueryClient();

  const invalidateBoardSnapshot = async () => {
    await queryClient.invalidateQueries({ queryKey: boardSnapshotQueryKey });
  };

  return {
    createTask: useMutation({
      mutationFn: (task: TaskDraft) => createTask(task),
      onSuccess: invalidateBoardSnapshot,
    }),
    moveTask: useMutation({
      mutationFn: (input: MoveTaskInput) => moveTask(input),
      onSuccess: invalidateBoardSnapshot,
    }),
    archiveTask: useMutation({
      mutationFn: (id: string) => archiveTask(id),
      onSuccess: invalidateBoardSnapshot,
    }),
    restoreTask: useMutation({
      mutationFn: (id: string) => restoreTask(id),
      onSuccess: invalidateBoardSnapshot,
    }),
  };
}
