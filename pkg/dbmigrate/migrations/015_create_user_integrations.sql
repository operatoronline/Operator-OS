-- User integrations: tracks per-user integration connection state
CREATE TABLE IF NOT EXISTS user_integrations (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    integration_id  TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    config          TEXT DEFAULT '{}',
    scopes          TEXT DEFAULT '[]',
    error_message   TEXT DEFAULT '',
    last_used_at    TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE(user_id, integration_id)
);

CREATE INDEX IF NOT EXISTS idx_user_integrations_user ON user_integrations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_integrations_status ON user_integrations(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_integrations_integration ON user_integrations(integration_id);
