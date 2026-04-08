-- Workspaces
CREATE TABLE IF NOT EXISTS workspaces (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata    JSONB NOT NULL DEFAULT '{}',
    archived_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API Keys (hash-only storage)
CREATE TABLE IF NOT EXISTS api_keys (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    name         TEXT NOT NULL DEFAULT '',
    key_hash     TEXT NOT NULL,
    key_prefix   TEXT NOT NULL,
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_workspace ON api_keys(workspace_id);

-- Add workspace_id to all resource tables
ALTER TABLE agents ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT '' REFERENCES workspaces(id);
ALTER TABLE environments ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT '' REFERENCES workspaces(id);
ALTER TABLE skills ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT '' REFERENCES workspaces(id);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT '' REFERENCES workspaces(id);
ALTER TABLE llm_request_logs ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT '' REFERENCES workspaces(id);

-- Rebuild usage_daily with workspace_id
ALTER TABLE usage_daily DROP CONSTRAINT IF EXISTS usage_daily_pkey;
ALTER TABLE usage_daily ADD COLUMN IF NOT EXISTS workspace_id TEXT NOT NULL DEFAULT '' REFERENCES workspaces(id);
ALTER TABLE usage_daily ADD CONSTRAINT usage_daily_pkey PRIMARY KEY (day, model, workspace_id);

-- Indexes for workspace filtering
CREATE INDEX IF NOT EXISTS idx_agents_workspace ON agents(workspace_id);
CREATE INDEX IF NOT EXISTS idx_environments_workspace ON environments(workspace_id);
CREATE INDEX IF NOT EXISTS idx_skills_workspace ON skills(workspace_id);
CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_llm_logs_workspace ON llm_request_logs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_usage_daily_workspace ON usage_daily(workspace_id);

-- Skills: change unique name index to per-workspace uniqueness
DROP INDEX IF EXISTS idx_skills_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_skills_workspace_name ON skills(workspace_id, name)
