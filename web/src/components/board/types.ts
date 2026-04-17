import type { TaskStatus } from "../../lib/api";

export type BoardLaneDescriptor = {
  label: string;
  status: TaskStatus;
};

export type MoveTarget = {
  label: string;
  status: TaskStatus;
};

export type MoveTaskRequest = {
  anchorTaskId?: string;
  id: string;
  placeAfter?: boolean;
  status: TaskStatus;
};

export type FocusTarget =
  | { kind: "archived"; id: string }
  | { kind: "task"; id: string }
  | { kind: "title" };
