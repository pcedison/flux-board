export type AuthSession = {
  authenticated: boolean;
  expiresAt: number;
};

export type TaskStatus = "queued" | "active" | "done";
export type TaskPriority = "medium" | "high" | "critical";

export type Task = {
  id: string;
  title: string;
  note: string;
  due: string;
  priority: TaskPriority;
  status: TaskStatus;
  sort_order: number;
  lastUpdated: number;
};

export type ArchivedTask = {
  id: string;
  title: string;
  note: string;
  due: string;
  priority: TaskPriority;
  status: TaskStatus;
  sort_order: number;
  archivedAt: number;
};

export type BoardSnapshot = {
  archived: ArchivedTask[];
  session: AuthSession | null;
  tasks: Task[];
};

type ErrorEnvelope = {
  error?: string;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export function isUnauthorizedApiError(error: unknown) {
  return error instanceof ApiError && error.status === 401;
}

type LoginResponse = {
  expiresAt?: number;
  ok?: boolean;
};

export type TaskDraft = {
  due: string;
  note?: string;
  priority: TaskPriority;
  title: string;
};

export type MoveTaskInput = {
  anchorTaskId?: string;
  id: string;
  placeAfter?: boolean;
  status: TaskStatus;
};

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers,
  });

  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ErrorEnvelope;
    throw new ApiError(body.error ?? `Request failed with status ${response.status}`, response.status);
  }

  return (await response.json()) as T;
}

async function apiFetchVoid(path: string, init?: RequestInit): Promise<void> {
  const headers = new Headers(init?.headers);
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers,
  });

  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ErrorEnvelope;
    throw new ApiError(body.error ?? `Request failed with status ${response.status}`, response.status);
  }
}

export async function fetchAuthSession(): Promise<AuthSession | null> {
  try {
    return await apiFetch<AuthSession>("/api/auth/me");
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      return null;
    }
    throw error;
  }
}

export async function loginWithPassword(password: string): Promise<AuthSession> {
  const body = await apiFetch<LoginResponse>("/api/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ password }),
  });

  return {
    authenticated: true,
    expiresAt: body.expiresAt ?? Date.now(),
  };
}

export function logout(): Promise<void> {
  return apiFetchVoid("/api/auth/logout", {
    method: "POST",
  });
}

export function createTask(task: TaskDraft): Promise<Task> {
  return apiFetch<Task>("/api/tasks", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      id: newTaskID(),
      title: task.title,
      note: task.note ?? "",
      due: task.due,
      priority: task.priority,
    }),
  });
}

export function moveTask(input: MoveTaskInput): Promise<Task> {
  return apiFetch<Task>(`/api/tasks/${input.id}/reorder`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      status: input.status,
      anchorTaskId: input.anchorTaskId,
      placeAfter: input.placeAfter,
    }),
  });
}

export function archiveTask(id: string): Promise<ArchivedTask> {
  return apiFetch<ArchivedTask>(`/api/tasks/${id}`, {
    method: "DELETE",
  });
}

export function restoreTask(id: string): Promise<Task> {
  return apiFetch<Task>(`/api/archived/${id}/restore`, {
    method: "POST",
  });
}

async function fetchTasks(): Promise<Task[]> {
  const body = await apiFetch<{ tasks?: Task[] }>("/api/tasks");
  return body.tasks ?? [];
}

async function fetchArchivedTasks(): Promise<ArchivedTask[]> {
  const body = await apiFetch<{ tasks?: ArchivedTask[] }>("/api/archived");
  return body.tasks ?? [];
}

export async function fetchBoardSnapshot(): Promise<BoardSnapshot> {
  const session = await fetchAuthSession();
  if (!session) {
    return {
      session: null,
      tasks: [],
      archived: [],
    };
  }

  const boardData = await Promise.all([fetchTasks(), fetchArchivedTasks()]).catch((error) => {
    if (isUnauthorizedApiError(error)) {
      return null;
    }
    throw error;
  });
  if (!boardData) {
    return {
      session: null,
      tasks: [],
      archived: [],
    };
  }
  const [tasks, archived] = boardData;
  return { session, tasks, archived };
}

function newTaskID(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `task-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`;
}
