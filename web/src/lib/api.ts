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

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

type LoginResponse = {
  expiresAt?: number;
  ok?: boolean;
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

  const [tasks, archived] = await Promise.all([fetchTasks(), fetchArchivedTasks()]);
  return { session, tasks, archived };
}
