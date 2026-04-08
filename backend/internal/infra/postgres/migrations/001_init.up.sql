-- Environments
CREATE TABLE IF NOT EXISTS environments (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    config      JSONB NOT NULL DEFAULT '{}',
    metadata    JSONB NOT NULL DEFAULT '{}',
    archived_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agents
CREATE TABLE IF NOT EXISTS agents (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    model       JSONB NOT NULL,
    system      TEXT NOT NULL DEFAULT '',
    tools       JSONB NOT NULL DEFAULT '[]',
    mcp_servers JSONB NOT NULL DEFAULT '[]',
    skills      JSONB NOT NULL DEFAULT '[]',
    metadata    JSONB NOT NULL DEFAULT '{}',
    version     INTEGER NOT NULL DEFAULT 1,
    archived_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    agent_snapshot  JSONB NOT NULL,
    environment_id  TEXT NOT NULL REFERENCES environments(id),
    title           TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'idle'
                        CHECK (status IN ('running', 'idle', 'terminated', 'rescheduling')),
    stats           JSONB NOT NULL DEFAULT '{}',
    usage           JSONB NOT NULL DEFAULT '{}',
    resources       JSONB NOT NULL DEFAULT '[]',
    metadata        JSONB NOT NULL DEFAULT '{}',
    vault_ids       JSONB NOT NULL DEFAULT '[]',
    archived_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_agent_id ON sessions((agent_snapshot->>'id'));
CREATE INDEX IF NOT EXISTS idx_sessions_environment ON sessions(environment_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);

-- Events
CREATE TABLE IF NOT EXISTS events (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL REFERENCES sessions(id),
    type         TEXT NOT NULL,
    payload      JSONB NOT NULL DEFAULT '{}',
    processed_at TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_events_session_created ON events(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_events_session_type ON events(session_id, type);
