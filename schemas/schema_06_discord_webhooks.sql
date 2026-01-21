-- ============================================================================
-- DISCORD WEBHOOKS SCHEMA
-- ============================================================================

-- Discord Webhooks (webhook management for Discord notifications)
CREATE TABLE IF NOT EXISTS discord_webhooks (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    name TEXT NOT NULL,
    "webhookUrl" TEXT NOT NULL,
    "webhookId" TEXT NOT NULL UNIQUE,
    
    type TEXT NOT NULL DEFAULT 'server_events',
    scope TEXT NOT NULL DEFAULT 'account',
    
    "serverId" TEXT REFERENCES servers(id) ON DELETE CASCADE,
    
    "isActive" BOOLEAN DEFAULT true,
    
    "notifyOnServerStart" BOOLEAN DEFAULT true,
    "notifyOnServerStop" BOOLEAN DEFAULT true,
    "notifyOnServerCrash" BOOLEAN DEFAULT true,
    "notifyOnBackupComplete" BOOLEAN DEFAULT true,
    "notifyOnBackupFailed" BOOLEAN DEFAULT true,
    "notifyOnConsoleOutput" BOOLEAN DEFAULT false,
    "notifyOnPlayerJoin" BOOLEAN DEFAULT false,
    "notifyOnPlayerLeave" BOOLEAN DEFAULT false,
    
    "customMessage" TEXT,
    
    "lastUsedAt" TIMESTAMP,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_discord_webhooks_user_id ON discord_webhooks("userId");
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_server_id ON discord_webhooks("serverId");
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_webhook_id ON discord_webhooks("webhookId");
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_type ON discord_webhooks(type);
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_is_active ON discord_webhooks("isActive");
