import type {
  Agent,
  Session,
  Environment,
  SessionEvent,
  ListResponse,
  UsageResponse,
  LogsResponse,
  LoginResult,
  RegisterResult,
  WorkspaceInfo,
  FileMetadata,
  SessionResource,
  Skill,
} from "./types";

const API_BASE =
  typeof window !== "undefined"
    ? (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080")
    : "http://localhost:8080";

// ── Auth token / workspace getters (read from localStorage) ──────────────

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

function getWorkspaceId(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("workspace_id");
}

// ── Core fetch helpers ───────────────────────────────────────────────────

function authHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  const wsId = getWorkspaceId();
  if (wsId) {
    headers["X-Workspace-Id"] = wsId;
  }
  return headers;
}

async function fetchJSON<T>(
  path: string,
  signal?: AbortSignal
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: authHeaders(),
    signal,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  return res.json();
}

async function postJSON<T>(
  path: string,
  body: unknown,
  signal?: AbortSignal
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify(body),
    signal,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

async function deleteJSON<T>(
  path: string,
  signal?: AbortSignal
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "DELETE",
    headers: authHeaders(),
    signal,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

// ── Auth (public — no token needed) ──────────────────────────────────────

export function login(email: string, password: string): Promise<LoginResult> {
  return postJSON("/v1/auth/login", { email, password });
}

export function register(
  email: string,
  password: string,
  name: string
): Promise<RegisterResult> {
  return postJSON("/v1/auth/register", { email, password, name });
}

export function listMyWorkspaces(): Promise<{ data: WorkspaceInfo[] }> {
  return fetchJSON("/v1/auth/workspaces");
}

// ── Agents ───────────────────────────────────────────────────────────────

export function listAgents(
  params?: {
    limit?: number;
    page?: string;
    include_archived?: boolean;
  },
  signal?: AbortSignal
): Promise<ListResponse<Agent>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.include_archived) search.set("include_archived", "true");
  const qs = search.toString();
  return fetchJSON(`/v1/agents${qs ? `?${qs}` : ""}`, signal);
}

export function getAgent(id: string, signal?: AbortSignal): Promise<Agent> {
  return fetchJSON(`/v1/agents/${id}`, signal);
}

export function createAgent(body: unknown): Promise<Agent> {
  return postJSON("/v1/agents", body);
}

export function updateAgent(
  id: string,
  body: {
    name?: string;
    description?: string;
    model?: string | { id: string; speed?: string };
    system?: string;
    tools?: unknown[];
    mcp_servers?: unknown[];
    skills?: unknown[];
    metadata?: Record<string, string>;
  }
): Promise<Agent> {
  return postJSON(`/v1/agents/${id}`, body);
}

export function archiveAgent(id: string): Promise<Agent> {
  return postJSON(`/v1/agents/${id}/archive`, {});
}

// ── Sessions ─────────────────────────────────────────────────────────────

export function listSessions(
  params?: {
    limit?: number;
    page?: string;
    agent_id?: string;
    include_archived?: boolean;
  },
  signal?: AbortSignal
): Promise<ListResponse<Session>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.agent_id) search.set("agent_id", params.agent_id);
  if (params?.include_archived) search.set("include_archived", "true");
  const qs = search.toString();
  return fetchJSON(`/v1/sessions${qs ? `?${qs}` : ""}`, signal);
}

export function getSession(
  id: string,
  signal?: AbortSignal
): Promise<Session> {
  return fetchJSON(`/v1/sessions/${id}`, signal);
}

export function listSessionEvents(
  sessionId: string,
  params?: { limit?: number; page?: string; order?: string },
  signal?: AbortSignal
): Promise<ListResponse<SessionEvent>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.order) search.set("order", params.order);
  const qs = search.toString();
  return fetchJSON(
    `/v1/sessions/${sessionId}/events${qs ? `?${qs}` : ""}`,
    signal
  );
}

// ── Environments ─────────────────────────────────────────────────────────

export function listEnvironments(
  params?: {
    limit?: number;
    page?: string;
    include_archived?: boolean;
  },
  signal?: AbortSignal
): Promise<ListResponse<Environment>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.include_archived) search.set("include_archived", "true");
  const qs = search.toString();
  return fetchJSON(`/v1/environments${qs ? `?${qs}` : ""}`, signal);
}

export function getEnvironment(
  id: string,
  signal?: AbortSignal
): Promise<Environment> {
  return fetchJSON(`/v1/environments/${id}`, signal);
}

export function createEnvironment(body: {
  name: string;
  description?: string;
  config?: {
    type: "cloud";
    networking?: {
      type: "unrestricted" | "limited";
      allow_mcp_servers?: boolean;
      allow_package_managers?: boolean;
      allowed_hosts?: string[];
    };
    packages?: {
      apt?: string[];
      cargo?: string[];
      gem?: string[];
      go?: string[];
      npm?: string[];
      pip?: string[];
    };
  };
}): Promise<Environment> {
  return postJSON("/v1/environments", body);
}

export function updateEnvironment(
  id: string,
  body: {
    name?: string;
    description?: string;
    config?: {
      type: "cloud";
      networking?: {
        type: "unrestricted" | "limited";
        allow_mcp_servers?: boolean;
        allow_package_managers?: boolean;
        allowed_hosts?: string[];
      };
      packages?: {
        apt?: string[];
        cargo?: string[];
        gem?: string[];
        go?: string[];
        npm?: string[];
        pip?: string[];
      };
    };
  }
): Promise<Environment> {
  return postJSON(`/v1/environments/${id}`, body);
}

export function archiveEnvironment(id: string): Promise<Environment> {
  return postJSON(`/v1/environments/${id}/archive`, {});
}

export function deleteEnvironment(
  id: string
): Promise<{ id: string; type: string }> {
  return deleteJSON(`/v1/environments/${id}`);
}

export function archiveSession(id: string): Promise<Session> {
  return postJSON(`/v1/sessions/${id}/archive`, {});
}

// ── Sessions (create / events) ──────────────────────────────────────────

export function createSession(body: {
  agent: string | { id: string; type: "agent"; version?: number };
  environment_id: string;
  title?: string;
  metadata?: Record<string, string>;
  resources?: readonly {
    type: "file";
    file_id: string;
    mount_path?: string;
  }[];
}): Promise<Session> {
  return postJSON("/v1/sessions", body);
}

export function sendSessionEvents(
  sessionId: string,
  events: readonly { type: string; content?: string; [key: string]: unknown }[]
): Promise<unknown> {
  return postJSON(`/v1/sessions/${sessionId}/events`, { events });
}

// ── Files ───────────────────────────────────────────────────────────

function authHeadersNoContentType(): Record<string, string> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  const wsId = getWorkspaceId();
  if (wsId) {
    headers["X-Workspace-Id"] = wsId;
  }
  return headers;
}

export async function uploadFile(file: File): Promise<FileMetadata> {
  const formData = new FormData();
  formData.append("file", file);
  const res = await fetch(`${API_BASE}/v1/files`, {
    method: "POST",
    headers: authHeadersNoContentType(),
    body: formData,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

export function listFiles(
  params?: {
    limit?: number;
    page?: string;
  },
  signal?: AbortSignal
): Promise<ListResponse<FileMetadata>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  const qs = search.toString();
  return fetchJSON(`/v1/files${qs ? `?${qs}` : ""}`, signal);
}

export function getFileMetadata(id: string): Promise<FileMetadata> {
  return fetchJSON(`/v1/files/${id}`);
}

export async function downloadFile(id: string): Promise<Blob> {
  const res = await fetch(`${API_BASE}/v1/files/${id}/content`, {
    headers: authHeaders(),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.blob();
}

export function deleteFile(id: string): Promise<{ id: string; type: string }> {
  return deleteJSON(`/v1/files/${id}`);
}

// ── Session Resources ───────────────────────────────────────────────

export function addSessionResource(
  sessionId: string,
  body: { type: "file"; file_id: string; mount_path?: string }
): Promise<SessionResource> {
  return postJSON(`/v1/sessions/${sessionId}/resources`, body);
}

export function listSessionResources(
  sessionId: string,
  params?: { limit?: number; page?: string }
): Promise<ListResponse<SessionResource>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  const qs = search.toString();
  return fetchJSON(
    `/v1/sessions/${sessionId}/resources${qs ? `?${qs}` : ""}`
  );
}

export function deleteSessionResource(
  sessionId: string,
  resourceId: string
): Promise<{ id: string; type: string }> {
  return deleteJSON(`/v1/sessions/${sessionId}/resources/${resourceId}`);
}

// ── Skills ───────────────────────────────────────────────────────────────

export function listSkills(
  params?: {
    limit?: number;
    page?: string;
  },
  signal?: AbortSignal
): Promise<ListResponse<Skill>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  const qs = search.toString();
  return fetchJSON(`/v1/skills${qs ? `?${qs}` : ""}`, signal);
}

export function getSkill(id: string): Promise<Skill> {
  return fetchJSON(`/v1/skills/${id}`);
}

export function createSkillFromMarkdown(content: string): Promise<Skill> {
  return postJSON("/v1/skills", { content });
}

export async function createSkillFromZip(file: File): Promise<Skill> {
  const formData = new FormData();
  formData.append("file", file);
  const res = await fetch(`${API_BASE}/v1/skills`, {
    method: "POST",
    headers: authHeadersNoContentType(),
    body: formData,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

export function deleteSkill(id: string): Promise<{ id: string; deleted: boolean }> {
  return deleteJSON(`/v1/skills/${id}`);
}

// ── Analytics ────────────────────────────────────────────────────────────

export function getUsage(
  params: {
    from?: string;
    to?: string;
    model?: string;
  },
  signal?: AbortSignal
): Promise<UsageResponse> {
  const search = new URLSearchParams();
  if (params.from) search.set("from", params.from);
  if (params.to) search.set("to", params.to);
  if (params.model) search.set("model", params.model);
  const qs = search.toString();
  return fetchJSON(`/v1/analytics/usage${qs ? `?${qs}` : ""}`, signal);
}

export function getLogs(
  params: {
    limit?: number;
    page?: string;
  },
  signal?: AbortSignal
): Promise<LogsResponse> {
  const search = new URLSearchParams();
  if (params.limit) search.set("limit", String(params.limit));
  if (params.page) search.set("page", params.page);
  const qs = search.toString();
  return fetchJSON(`/v1/analytics/logs${qs ? `?${qs}` : ""}`, signal);
}
