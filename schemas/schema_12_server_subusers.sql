-- ============================================================================
-- SERVER SUBUSERS - User-Server Relationships & Permissions
-- ============================================================================

-- Track user access to servers (including owners and subusers)
-- This table captures the many-to-many relationship between users and servers
-- with permission tracking for access control and auditing
CREATE TABLE IF NOT EXISTS server_subusers (
    id TEXT PRIMARY KEY,
    "serverId" TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Permissions can be:
    -- 1. Pterodactyl: ["control.console", "control.start", "file.read", "user.create"]
    -- 2. VPS/Linux: ["ssh", "sudo", "file_manager", "package_manager"]
    -- 3. Email: ["manage_mailboxes", "manage_domains", "manage_forwarding"]
    permissions TEXT[] DEFAULT '{}',
    
    -- Access level for non-permission-based systems
    -- Values: owner, admin, user, viewer, billing_only
    "accessLevel" TEXT DEFAULT 'user',
    
    -- Type-specific access configuration stored as JSON
    -- Examples:
    -- pterodactyl: {"canCreateSubusers": false, "canDeleteServer": false}
    -- vps: {"sshKeyOnly": true, "ipWhitelistEnabled": false, "ipWhitelist": ["1.2.3.4"]}
    -- email: {"canCreateMailboxes": true, "maxMailboxes": 10, "maxStorage": 100}
    "accessConfig" JSONB DEFAULT '{}',
    
    -- Metadata
    "isOwner" BOOLEAN DEFAULT false,
    "addedAt" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "lastSyncedAt" TIMESTAMP,
    
    -- Prevent duplicate user-server relationships
    CONSTRAINT server_subusers_unique UNIQUE ("serverId", "userId")
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_server_subusers_server ON server_subusers("serverId");
CREATE INDEX IF NOT EXISTS idx_server_subusers_user ON server_subusers("userId");
CREATE INDEX IF NOT EXISTS idx_server_subusers_owner ON server_subusers("isOwner") WHERE "isOwner" = true;
CREATE INDEX IF NOT EXISTS idx_server_subusers_access_level ON server_subusers("accessLevel");

-- Comments for documentation
COMMENT ON TABLE server_subusers IS 'User-server relationships including owners and subusers with flexible permission/access control';
COMMENT ON COLUMN server_subusers.permissions IS 'Array of permission strings (format varies by serverType: Pterodactyl, VPS, email, etc.)';
COMMENT ON COLUMN server_subusers."accessLevel" IS 'Hierarchical access level: owner, admin, user, viewer, billing_only';
COMMENT ON COLUMN server_subusers."accessConfig" IS 'Type-specific access configuration for additional control rules';
COMMENT ON COLUMN server_subusers."isOwner" IS 'True if this user is the primary owner of the server';
COMMENT ON COLUMN server_subusers."lastSyncedAt" IS 'Last time this relationship was synced from the control panel';
