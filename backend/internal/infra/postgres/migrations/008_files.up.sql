-- Files: workspace-scoped file storage
CREATE TABLE IF NOT EXISTS files (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL REFERENCES workspaces(id),
    filename      TEXT NOT NULL,
    mime_type     TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes    BIGINT NOT NULL DEFAULT 0,
    storage_path  TEXT NOT NULL,
    downloadable  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_files_workspace ON files(workspace_id);
CREATE INDEX IF NOT EXISTS idx_files_workspace_created ON files(workspace_id, created_at DESC);

-- Session resources: structured resource records replacing raw JSONB
CREATE TABLE IF NOT EXISTS session_resources (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('file', 'github_repository')),
    file_id     TEXT REFERENCES files(id),
    mount_path  TEXT NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_session_resources_session ON session_resources(session_id);
CREATE INDEX IF NOT EXISTS idx_session_resources_file ON session_resources(file_id);
