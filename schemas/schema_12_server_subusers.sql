-- ============================================================================
-- SERVER SUBUSERS - User-Server Relationships & Permissions
-- ============================================================================

-- Track user access to servers (including owners and subusers)
-- This table captures the many-to-many relationship between users and servers
-- with permission tracking for access control and auditing
CREATE TABLE IF NOT EXISTS server_subusers (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Permissions array from Pterodactyl Client API
    -- Examples: ["control.console", "control.start", "file.read", "user.create"]
    permissions TEXT[] DEFAULT '{}',
    
    -- Metadata
    is_owner BOOLEAN DEFAULT false,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP,
    
    -- Prevent duplicate user-server relationships
    CONSTRAINT server_subusers_unique UNIQUE (server_id, user_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_server_subusers_server ON server_subusers(server_id);
CREATE INDEX IF NOT EXISTS idx_server_subusers_user ON server_subusers(user_id);
CREATE INDEX IF NOT EXISTS idx_server_subusers_owner ON server_subusers(is_owner) WHERE is_owner = true;

-- Comments for documentation
COMMENT ON TABLE server_subusers IS 'User-server relationships including owners and subusers with permissions';
COMMENT ON COLUMN server_subusers.permissions IS 'Array of Pterodactyl permission strings for this user on this server';
COMMENT ON COLUMN server_subusers.is_owner IS 'True if this user is the primary owner of the server';
COMMENT ON COLUMN server_subusers.last_synced_at IS 'Last time this relationship was synced from Pterodactyl panel';
