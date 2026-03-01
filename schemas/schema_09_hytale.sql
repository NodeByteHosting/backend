-- Hytale OAuth Tokens
-- Stores OAuth tokens obtained from Hytale OAuth provider
CREATE TABLE IF NOT EXISTS hytale_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    access_token_expiry TIMESTAMP NOT NULL,
    profile_uuid UUID,
    scope TEXT NOT NULL DEFAULT 'openid offline auth:server',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_refreshed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hytale_oauth_tokens_account_id ON hytale_oauth_tokens(account_id);
CREATE INDEX IF NOT EXISTS idx_hytale_oauth_tokens_access_token_expiry ON hytale_oauth_tokens(access_token_expiry);

-- Hytale Game Sessions
-- Stores active game sessions for Hytale servers
CREATE TABLE IF NOT EXISTS hytale_game_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    profile_uuid UUID NOT NULL,
    server_id TEXT,
    session_token TEXT NOT NULL,
    identity_token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT hytale_game_sessions_account_profile_key UNIQUE (account_id, profile_uuid)
);

CREATE INDEX IF NOT EXISTS idx_hytale_game_sessions_account_id ON hytale_game_sessions(account_id);
CREATE INDEX IF NOT EXISTS idx_hytale_game_sessions_expires_at ON hytale_game_sessions(expires_at);
