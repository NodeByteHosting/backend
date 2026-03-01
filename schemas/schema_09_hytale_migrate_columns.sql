-- Migration: rename camelCase columns to snake_case in hytale tables
-- Safe to run multiple times (checks column existence before renaming)

DO $$
BEGIN
    -- hytale_oauth_tokens renames
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='accountId') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "accountId" TO account_id;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='accessToken') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "accessToken" TO access_token;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='refreshToken') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "refreshToken" TO refresh_token;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='accessTokenExpiry') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "accessTokenExpiry" TO access_token_expiry;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='profileUuid') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "profileUuid" TO profile_uuid;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='createdAt') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "createdAt" TO created_at;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='updatedAt') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "updatedAt" TO updated_at;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_oauth_tokens' AND column_name='lastRefreshedAt') THEN
        ALTER TABLE hytale_oauth_tokens RENAME COLUMN "lastRefreshedAt" TO last_refreshed_at;
    END IF;

    -- hytale_game_sessions renames
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='accountId') THEN
        ALTER TABLE hytale_game_sessions RENAME COLUMN "accountId" TO account_id;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='profileUuid') THEN
        ALTER TABLE hytale_game_sessions RENAME COLUMN "profileUuid" TO profile_uuid;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='sessionToken') THEN
        ALTER TABLE hytale_game_sessions RENAME COLUMN "sessionToken" TO session_token;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='identityToken') THEN
        ALTER TABLE hytale_game_sessions RENAME COLUMN "identityToken" TO identity_token;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='expiresAt') THEN
        ALTER TABLE hytale_game_sessions RENAME COLUMN "expiresAt" TO expires_at;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='createdAt') THEN
        ALTER TABLE hytale_game_sessions RENAME COLUMN "createdAt" TO created_at;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='updatedAt') THEN
        ALTER TABLE hytale_game_sessions RENAME COLUMN "updatedAt" TO updated_at;
    END IF;

    -- Add server_id if missing
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='hytale_game_sessions' AND column_name='server_id') THEN
        ALTER TABLE hytale_game_sessions ADD COLUMN server_id TEXT;
    END IF;
END $$;

-- Recreate indexes using the new snake_case column names (DROP IF EXISTS first)
DROP INDEX IF EXISTS idx_hytale_oauth_tokens_account_id;
DROP INDEX IF EXISTS idx_hytale_oauth_tokens_access_token_expiry;
DROP INDEX IF EXISTS idx_hytale_game_sessions_account_id;
DROP INDEX IF EXISTS idx_hytale_game_sessions_expires_at;

CREATE INDEX IF NOT EXISTS idx_hytale_oauth_tokens_account_id ON hytale_oauth_tokens(account_id);
CREATE INDEX IF NOT EXISTS idx_hytale_oauth_tokens_access_token_expiry ON hytale_oauth_tokens(access_token_expiry);
CREATE INDEX IF NOT EXISTS idx_hytale_game_sessions_account_id ON hytale_game_sessions(account_id);
CREATE INDEX IF NOT EXISTS idx_hytale_game_sessions_expires_at ON hytale_game_sessions(expires_at);
