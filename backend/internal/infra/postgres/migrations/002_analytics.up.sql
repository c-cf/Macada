-- llm_request_logs: one row per model request (populated by session runner)
CREATE TABLE IF NOT EXISTS llm_request_logs (
    id                    TEXT PRIMARY KEY,
    session_id            TEXT NOT NULL REFERENCES sessions(id),
    agent_id              TEXT NOT NULL,
    model                 TEXT NOT NULL,
    input_tokens          BIGINT NOT NULL DEFAULT 0,
    output_tokens         BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens     BIGINT NOT NULL DEFAULT 0,
    cache_creation_tokens BIGINT NOT NULL DEFAULT 0,
    latency_ms            INTEGER NOT NULL DEFAULT 0,
    is_error              BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_llm_logs_created ON llm_request_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_llm_logs_session ON llm_request_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_llm_logs_model ON llm_request_logs(model);

-- usage_daily: pre-aggregated daily usage
CREATE TABLE IF NOT EXISTS usage_daily (
    day                   DATE NOT NULL,
    model                 TEXT NOT NULL DEFAULT '',
    input_tokens          BIGINT NOT NULL DEFAULT 0,
    output_tokens         BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens     BIGINT NOT NULL DEFAULT 0,
    cache_creation_tokens BIGINT NOT NULL DEFAULT 0,
    request_count         BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, model)
);
