-- ============================================================================
-- HYTALE SERVER LINKAGE
-- ============================================================================
-- Links Hytale game sessions to specific servers for environment variable updates

-- Add server_id column to hytale_game_sessions if not exists
ALTER TABLE hytale_game_sessions 
ADD COLUMN IF NOT EXISTS server_id TEXT REFERENCES servers(id) ON DELETE CASCADE;

-- Create index for efficient server-based lookups
CREATE INDEX IF NOT EXISTS idx_hytale_game_sessions_server_id ON hytale_game_sessions(server_id);

-- Comment explaining the relationship
COMMENT ON COLUMN hytale_game_sessions.server_id IS 'Links game session to specific Pterodactyl server for automatic token push';
