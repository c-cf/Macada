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

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: authHeaders(),
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  return res.json();
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

async function deleteJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "DELETE",
    headers: authHeaders(),
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

export function listAgents(params?: {
  limit?: number;
  page?: string;
  include_archived?: boolean;
}): Promise<ListResponse<Agent>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.include_archived) search.set("include_archived", "true");
  const qs = search.toString();
  return fetchJSON(`/v1/agents${qs ? `?${qs}` : ""}`);
}

export function getAgent(id: string): Promise<Agent> {
  return fetchJSON(`/v1/agents/${id}`);
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

export function listSessions(params?: {
  limit?: number;
  page?: string;
  agent_id?: string;
  include_archived?: boolean;
}): Promise<ListResponse<Session>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.agent_id) search.set("agent_id", params.agent_id);
  if (params?.include_archived) search.set("include_archived", "true");
  const qs = search.toString();
  return fetchJSON(`/v1/sessions${qs ? `?${qs}` : ""}`);
}

export function getSession(id: string): Promise<Session> {
  return fetchJSON(`/v1/sessions/${id}`);
}

export function listSessionEvents(
  sessionId: string,
  params?: { limit?: number; page?: string; order?: string }
): Promise<ListResponse<SessionEvent>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.order) search.set("order", params.order);
  const qs = search.toString();
  return fetchJSON(`/v1/sessions/${sessionId}/events${qs ? `?${qs}` : ""}`);
}

// ── Environments ─────────────────────────────────────────────────────────

export function listEnvironments(params?: {
  limit?: number;
  page?: string;
  include_archived?: boolean;
}): Promise<ListResponse<Environment>> {
  const search = new URLSearchParams();
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.page) search.set("page", params.page);
  if (params?.include_archived) search.set("include_archived", "true");
  const qs = search.toString();
  return fetchJSON(`/v1/environments${qs ? `?${qs}` : ""}`);
}

export function getEnvironment(id: string): Promise<Environment> {
  return fetchJSON(`/v1/environments/${id}`);
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

// ── Sessions (create / events) ──────────────────────────────────────────

export function createSession(body: {
  agent: string | { id: string; type: "agent"; version?: number };
  environment_id: string;
  title?: string;
  metadata?: Record<string, string>;
}): Promise<Session> {
  return postJSON("/v1/sessions", body);
}

export function sendSessionEvents(
  sessionId: string,
  events: readonly { type: string; content?: string; [key: string]: unknown }[]
): Promise<unknown> {
  return postJSON(`/v1/sessions/${sessionId}/events`, { events });
}

// ── Analytics ────────────────────────────────────────────────────────────

export function getUsage(params: {
  from?: string;
  to?: string;
  model?: string;
}): Promise<UsageResponse> {
  const search = new URLSearchParams();
  if (params.from) search.set("from", params.from);
  if (params.to) search.set("to", params.to);
  if (params.model) search.set("model", params.model);
  const qs = search.toString();
  return fetchJSON(`/v1/analytics/usage${qs ? `?${qs}` : ""}`);
}

export function getLogs(params: {
  limit?: number;
  page?: string;
}): Promise<LogsResponse> {
  const search = new URLSearchParams();
  if (params.limit) search.set("limit", String(params.limit));
  if (params.page) search.set("page", params.page);
  const qs = search.toString();
  return fetchJSON(`/v1/analytics/logs${qs ? `?${qs}` : ""}`);
}
