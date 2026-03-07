-- 014_create_credential_vault.sql
-- Per-user per-integration encrypted credential vault for OAuth tokens.
CREATE TABLE IF NOT EXISTS credential_vault (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    provider_id     TEXT NOT NULL,
    encrypted_data  BLOB NOT NULL,
    encrypted       INTEGER NOT NULL DEFAULT 1,
    label           TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active',
    scopes          TEXT NOT NULL DEFAULT '',
    expires_at      TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    UNIQUE(user_id, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_credential_vault_user ON credential_vault(user_id);
CREATE INDEX IF NOT EXISTS idx_credential_vault_provider ON credential_vault(provider_id);
CREATE INDEX IF NOT EXISTS idx_credential_vault_user_status ON credential_vault(user_id, status);
