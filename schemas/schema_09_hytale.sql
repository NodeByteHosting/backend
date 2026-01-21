-- Hytale OAuth Tokens
-- Stores OAuth tokens obtained from Hytale OAuth provider
CREATE TABLE IF NOT EXISTS hytale_oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "accountId" UUID NOT NULL UNIQUE,
    "accessToken" TEXT NOT NULL,
    "refreshToken" TEXT NOT NULL,
    "accessTokenExpiry" TIMESTAMP NOT NULL,
    "profileUuid" UUID,
    scope TEXT NOT NULL DEFAULT 'openid offline auth:server',
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastRefreshedAt" TIMESTAMP,
    CONSTRAINT hytale_oauth_tokens_account_id_key UNIQUE ("accountId")
);

CREATE INDEX IF NOT EXISTS idx_hytale_oauth_tokens_account_id ON hytale_oauth_tokens("accountId");
CREATE INDEX IF NOT EXISTS idx_hytale_oauth_tokens_access_token_expiry ON hytale_oauth_tokens("accessTokenExpiry");

-- Hytale Game Sessions
-- Stores active game sessions for Hytale servers
CREATE TABLE IF NOT EXISTS hytale_game_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "accountId" UUID NOT NULL,
    "profileUuid" UUID NOT NULL,
    "sessionToken" TEXT NOT NULL,
    "identityToken" TEXT NOT NULL,
    "expiresAt" TIMESTAMP NOT NULL,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT hytale_game_sessions_account_profile_key UNIQUE ("accountId", "profileUuid")
);

CREATE INDEX IF NOT EXISTS idx_hytale_game_sessions_account_id ON hytale_game_sessions("accountId");
CREATE INDEX IF NOT EXISTS idx_hytale_game_sessions_expires_at ON hytale_game_sessions("expiresAt");
