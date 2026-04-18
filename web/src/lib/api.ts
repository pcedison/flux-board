export type AuthSession = {
  authenticated: boolean;
  expiresAt: number;
};

export type BootstrapStatus = {
  needsSetup: boolean;
};

export type AppStatusCheck = {
  name: string;
  ok: boolean;
  message: string;
};

export type AppStatus = {
  status: "ready" | "degraded";
  version: string;
  environment: string;
  needsSetup: boolean;
  archiveRetentionDays: number | null;
  runtimeArtifact: string;
  runtimeOwnershipPath: string;
  legacyRollbackPath: string;
  archiveCleanupEvery: string;
  sessionCleanupEvery: string;
  generatedAt: number;
  checks: AppStatusCheck[];
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

export type SessionInfo = {
  token: string;
  createdAt: number;
  expiresAt: number;
  lastSeenAt: number;
  clientIP: string;
  current: boolean;
};

export type AppSettings = {
  archiveRetentionDays: number | null;
};

export type ExportBundle = {
  version: string;
  exportedAt: number;
  settings: AppSettings;
  tasks: Task[];
  archived: ArchivedTask[];
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

export function isSetupRequiredApiError(error: unknown) {
  return error instanceof ApiError && error.status === 409 && error.message === "setup required";
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

export type TaskUpdateDraft = TaskDraft;

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

export function fetchBootstrapStatus(): Promise<BootstrapStatus> {
  return apiFetch<BootstrapStatus>("/api/bootstrap/status");
}

export function fetchAppStatus(): Promise<AppStatus> {
  return apiFetch<AppStatus>("/api/status");
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

export async function bootstrapWithPassword(password: string): Promise<AuthSession> {
  const body = await apiFetch<LoginResponse>("/api/bootstrap/setup", {
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

export function updateTask(id: string, task: TaskUpdateDraft): Promise<Task> {
  return apiFetch<Task>(`/api/tasks/${id}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
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

export function deleteArchivedTask(id: string): Promise<void> {
  return apiFetchVoid(`/api/archived/${id}`, {
    method: "DELETE",
  });
}

export function fetchSettings(): Promise<AppSettings> {
  return apiFetch<AppSettings>("/api/settings");
}

export function updateSettings(settings: AppSettings): Promise<AppSettings> {
  return apiFetch<AppSettings>("/api/settings", {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(settings),
  });
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return apiFetchVoid("/api/settings/password", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ currentPassword, newPassword }),
  });
}

export async function fetchSessions(): Promise<SessionInfo[]> {
  const body = await apiFetch<{ sessions?: SessionInfo[] }>("/api/settings/sessions");
  return body.sessions ?? [];
}

export function revokeSession(token: string): Promise<void> {
  return apiFetchVoid(`/api/settings/sessions/${token}`, {
    method: "DELETE",
  });
}

export function exportBoardData(): Promise<ExportBundle> {
  return apiFetch<ExportBundle>("/api/export");
}

export function importBoardData(bundle: ExportBundle): Promise<void> {
  return apiFetchVoid("/api/import", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(bundle),
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
