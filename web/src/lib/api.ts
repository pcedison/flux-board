export type AuthSession = {
  authenticated: boolean;
  expiresAt: number;
};

export type TaskStatus = "queued" | "active" | "done";

export type Task = {
  id: string;
  title: string;
  note: string;
  due: string;
  priority: "medium" | "high" | "critical";
  status: TaskStatus;
  sort_order: number;
  lastUpdated: number;
};

export type ArchivedTask = {
  id: string;
  title: string;
  note: string;
  due: string;
  priority: "medium" | "high" | "critical";
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

class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function apiFetch<T>(path: string): Promise<T> {
  const response = await fetch(path, {
    credentials: "include",
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ErrorEnvelope;
    throw new ApiError(body.error ?? `Request failed with status ${response.status}`, response.status);
  }

  return (await response.json()) as T;
}

async function fetchSession(): Promise<AuthSession | null> {
  try {
    return await apiFetch<AuthSession>("/api/auth/me");
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      return null;
    }
    throw error;
  }
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
  const session = await fetchSession();
  if (!session) {
    return {
      session: null,
      tasks: [],
      archived: [],
    };
  }

  const [tasks, archived] = await Promise.all([fetchTasks(), fetchArchivedTasks()]);
  return { session, tasks, archived };
}
