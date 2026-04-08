ALTER TABLE sessions ADD COLUMN IF NOT EXISTS sandbox_id TEXT;

CREATE TABLE IF NOT EXISTS sandboxes (
    id               TEXT PRIMARY KEY,
    session_id       TEXT NOT NULL REFERENCES sessions(id),
    container_id     TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'running', 'stopped', 'error')),
    container_ip     TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sandboxes_session ON sandboxes(session_id)
