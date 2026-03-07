CREATE TABLE IF NOT EXISTS oauth_states (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL,
    provider_id   TEXT NOT NULL,
    state         TEXT NOT NULL UNIQUE,
    code_verifier TEXT DEFAULT '',
    redirect_uri  TEXT DEFAULT '',
    scopes        TEXT DEFAULT '',
    created_at    TEXT NOT NULL,
    expires_at    TEXT NOT NULL,
    used          INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_state ON oauth_states(state);
CREATE INDEX IF NOT EXISTS idx_oauth_states_user_id ON oauth_states(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);
