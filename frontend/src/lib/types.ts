// API response types matching the Go backend domain models

export interface ModelConfig {
  readonly id: string;
  readonly speed?: string;
}

export interface Agent {
  readonly id: string;
  readonly name: string;
  readonly description: string;
  readonly model: ModelConfig;
  readonly system: string;
  readonly tools: unknown[];
  readonly mcp_servers: unknown[];
  readonly skills: unknown[];
  readonly metadata: Record<string, string>;
  readonly version: number;
  readonly archived_at: string | null;
  readonly created_at: string;
  readonly updated_at: string;
  readonly type: string;
}

export interface SessionStats {
  readonly active_seconds: number | null;
  readonly duration_seconds: number | null;
}

export interface SessionUsage {
  readonly input_tokens: number | null;
  readonly output_tokens: number | null;
  readonly cache_read_input_tokens: number | null;
  readonly cache_creation: {
    readonly ephemeral_1h_input_tokens: number | null;
    readonly ephemeral_5m_input_tokens: number | null;
  } | null;
}

export interface SessionAgent {
  readonly id: string;
  readonly version: number;
  readonly type: string;
  readonly name: string;
  readonly description: string;
  readonly model: ModelConfig;
  readonly system: string;
  readonly tools: unknown[];
  readonly mcp_servers: unknown[];
  readonly skills: unknown[];
}

export interface Session {
  readonly id: string;
  readonly agent: SessionAgent;
  readonly environment_id: string;
  readonly title: string;
  readonly status: "running" | "idle" | "terminated" | "rescheduling";
  readonly stats: SessionStats;
  readonly usage: SessionUsage;
  readonly resources: unknown[];
  readonly metadata: Record<string, string>;
  readonly vault_ids: string[];
  readonly archived_at: string | null;
  readonly created_at: string;
  readonly updated_at: string;
  readonly type: string;
}

export interface NetworkingConfig {
  readonly type: string;
  readonly allow_mcp_servers?: boolean;
  readonly allow_package_managers?: boolean;
  readonly allowed_hosts?: string[];
}

export interface EnvironmentConfig {
  readonly type: string;
  readonly networking: NetworkingConfig;
  readonly packages: {
    readonly apt: string[];
    readonly cargo: string[];
    readonly gem: string[];
    readonly go: string[];
    readonly npm: string[];
    readonly pip: string[];
    readonly type?: string;
  };
}

export interface Environment {
  readonly id: string;
  readonly name: string;
  readonly description: string;
  readonly config: EnvironmentConfig;
  readonly metadata: Record<string, string>;
  readonly archived_at: string | null;
  readonly created_at: string;
  readonly updated_at: string;
  readonly type: string;
}

// Files API types

export interface FileMetadata {
  readonly id: string;
  readonly filename: string;
  readonly mime_type: string;
  readonly size_bytes: number;
  readonly downloadable: boolean;
  readonly created_at: string;
  readonly type: "file";
}

export interface FileResource {
  readonly id: string;
  readonly session_id?: string;
  readonly file_id: string;
  readonly mount_path: string;
  readonly type: "file";
  readonly config?: unknown;
  readonly created_at: string;
  readonly updated_at: string;
}

export interface GitHubRepositoryResource {
  readonly id: string;
  readonly session_id?: string;
  readonly url: string;
  readonly mount_path: string;
  readonly type: "github_repository";
  readonly config?: unknown;
  readonly created_at: string;
  readonly updated_at: string;
}

export type SessionResource = FileResource | GitHubRepositoryResource;

export interface SessionEvent {
  readonly id: string;
  readonly type: string;
  readonly processed_at: string;
  readonly [key: string]: unknown;
}

export interface ListResponse<T> {
  readonly data: T[];
  readonly next_page: string | null;
}

// Auth types

export interface User {
  readonly id: string;
  readonly email: string;
  readonly name: string;
  readonly created_at: string;
  readonly updated_at: string;
  readonly type: string;
}

export interface WorkspaceInfo {
  readonly id: string;
  readonly name: string;
  readonly role: string;
}

export interface LoginResult {
  readonly user: User;
  readonly token: string;
  readonly expires_at: string;
  readonly workspaces: WorkspaceInfo[];
}

export interface RegisterResult {
  readonly user: User;
  readonly token: string;
  readonly expires_at: string;
  readonly workspaces: WorkspaceInfo[];
}

// Analytics types

export interface UsageDayData {
  readonly day: string;
  readonly model: string;
  readonly input_tokens: number;
  readonly output_tokens: number;
  readonly cache_read_tokens: number;
  readonly cache_creation_tokens: number;
  readonly request_count: number;
}

export interface UsageSummary {
  readonly total_input: number;
  readonly total_output: number;
  readonly total_requests: number;
}

export interface UsageResponse {
  readonly data: UsageDayData[];
  readonly summary: UsageSummary;
}

export interface LogEntry {
  readonly id: string;
  readonly session_id: string;
  readonly agent_id: string;
  readonly model: string;
  readonly input_tokens: number;
  readonly output_tokens: number;
  readonly cache_read_tokens: number;
  readonly cache_creation_tokens: number;
  readonly latency_ms: number;
  readonly is_error: boolean;
  readonly created_at: string;
}

export interface LogsResponse {
  readonly data: LogEntry[];
  readonly next_page: string | null;
}

// Skills API types

export interface Skill {
  readonly id: string;
  readonly name: string;
  readonly description: string;
  readonly license: string;
  readonly compatibility: string;
  readonly allowed_tools: string;
  readonly metadata: Record<string, string>;
  readonly content: string;
  readonly created_at: string;
  readonly updated_at: string;
  readonly type: "skill";
}
