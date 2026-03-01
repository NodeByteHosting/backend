-- ============================================================================
-- DISCORD WEBHOOKS SCHEMA
-- ============================================================================

-- Discord Webhooks (admin-managed webhook configurations for Discord notifications)
CREATE TABLE IF NOT EXISTS discord_webhooks (
    id TEXT PRIMARY KEY,

    name TEXT NOT NULL,
    "webhookUrl" TEXT NOT NULL,

    -- Type: GAME_SERVER | VPS | SYSTEM | BILLING | SECURITY | SUPPORT | CUSTOM
    type TEXT NOT NULL DEFAULT 'SYSTEM',
    -- Scope: ADMIN | USER | PUBLIC
    scope TEXT NOT NULL DEFAULT 'ADMIN',

    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,

    "testSuccessAt" TIMESTAMP,

    "createdAt" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discord_webhooks_type ON discord_webhooks(type);
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_scope ON discord_webhooks(scope);
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_enabled ON discord_webhooks(enabled);
