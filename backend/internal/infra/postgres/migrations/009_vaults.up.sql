CREATE TABLE IF NOT EXISTS vaults (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL REFERENCES workspaces(id),
    display_name  TEXT NOT NULL,
    metadata      JSONB NOT NULL DEFAULT '{}',
    archived_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vaults_workspace ON vaults(workspace_id);
CREATE INDEX IF NOT EXISTS idx_vaults_workspace_created ON vaults(workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS vault_secrets (
    vault_id        TEXT NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    encrypted_value BYTEA NOT NULL,
    nonce           BYTEA NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (vault_id, key)
);
