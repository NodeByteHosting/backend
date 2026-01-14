-- ============================================================================
-- DISCORD WEBHOOKS SCHEMA
-- ============================================================================

-- Discord Webhooks (webhook management for Discord notifications)
CREATE TABLE IF NOT EXISTS discord_webhooks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    name TEXT NOT NULL,
    webhook_url TEXT NOT NULL,
    webhook_id TEXT NOT NULL UNIQUE,
    
    type TEXT NOT NULL DEFAULT 'server_events',
    scope TEXT NOT NULL DEFAULT 'account',
    
    server_id TEXT REFERENCES servers(id) ON DELETE CASCADE,
    
    is_active BOOLEAN DEFAULT true,
    
    notify_on_server_start BOOLEAN DEFAULT true,
    notify_on_server_stop BOOLEAN DEFAULT true,
    notify_on_server_crash BOOLEAN DEFAULT true,
    notify_on_backup_complete BOOLEAN DEFAULT true,
    notify_on_backup_failed BOOLEAN DEFAULT true,
    notify_on_console_output BOOLEAN DEFAULT false,
    notify_on_player_join BOOLEAN DEFAULT false,
    notify_on_player_leave BOOLEAN DEFAULT false,
    
    custom_message TEXT,
    
    last_used_at TIMESTAMP,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_discord_webhooks_user_id ON discord_webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_server_id ON discord_webhooks(server_id);
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_webhook_id ON discord_webhooks(webhook_id);
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_type ON discord_webhooks(type);
CREATE INDEX IF NOT EXISTS idx_discord_webhooks_is_active ON discord_webhooks(is_active);
