-- Restore skills unique name index
DROP INDEX IF EXISTS idx_skills_workspace_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_skills_name ON skills(name);

-- Drop workspace indexes
DROP INDEX IF EXISTS idx_usage_daily_workspace;
DROP INDEX IF EXISTS idx_llm_logs_workspace;
DROP INDEX IF EXISTS idx_sessions_workspace;
DROP INDEX IF EXISTS idx_skills_workspace;
DROP INDEX IF EXISTS idx_environments_workspace;
DROP INDEX IF EXISTS idx_agents_workspace;

-- Rebuild usage_daily without workspace_id
ALTER TABLE usage_daily DROP CONSTRAINT IF EXISTS usage_daily_pkey;
ALTER TABLE usage_daily DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE usage_daily ADD CONSTRAINT usage_daily_pkey PRIMARY KEY (day, model);

-- Remove workspace_id from resource tables
ALTER TABLE llm_request_logs DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE skills DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE environments DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE agents DROP COLUMN IF EXISTS workspace_id;

-- Drop new tables
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS workspaces
