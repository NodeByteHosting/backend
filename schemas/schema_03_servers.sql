-- ============================================================================
-- SERVERS & RELATED TABLES
-- ============================================================================

-- Servers (game server instances and other hosting services)
CREATE TABLE IF NOT EXISTS servers (
    id TEXT PRIMARY KEY,
    
    -- Server type classification (game_server, vps, email, web_hosting, etc.)
    "serverType" TEXT NOT NULL DEFAULT 'game_server',
    
    -- Pterodactyl panel identifiers (nullable for non-game-server types)
    "pterodactylId" INTEGER UNIQUE,
    "virtfusionId" INTEGER UNIQUE,
    uuid TEXT UNIQUE, -- nullable for non-Pterodactyl servers
    "uuidShort" TEXT,
    "externalId" TEXT UNIQUE,
    
    -- Panel integration details (specific to serverType)
    "panelType" TEXT DEFAULT 'pterodactyl', -- pterodactyl, proxmox, cPanel, etc.
    
    -- Pterodactyl specific (for game_server type)
    "eggId" INTEGER REFERENCES eggs(id) ON DELETE SET NULL,
    "nestId" INTEGER REFERENCES nests(id) ON DELETE SET NULL,
    
    -- Core server info
    name TEXT NOT NULL,
    description TEXT,
    
    -- Server state
    status TEXT DEFAULT 'installing', -- installing, online, offline, suspended, error
    "isSuspended" BOOLEAN DEFAULT false,
    
    -- Product and location
    "productId" TEXT REFERENCES products(id) ON DELETE SET NULL,
    "ownerId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "nodeId" INTEGER REFERENCES nodes(id) ON DELETE SET NULL, -- nullable for non-panel servers
    
    -- Server-type-specific configuration stored as JSON
    -- Examples:
    -- game_server: {"autoRestart": true, "backupSchedule": "daily"}
    -- vps: {"osType": "Ubuntu", "cpuLimit": 2, "ramLimit": 4096}
    -- email: {"domainName": "example.com", "mailServerType": "postfix"}
    config JSONB DEFAULT '{}',
    
    -- Timestamps
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "installedAt" TIMESTAMP,
    "lastSyncedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_servers_uuid ON servers(uuid);
CREATE INDEX IF NOT EXISTS idx_servers_pterodactyl_id ON servers("pterodactylId");
CREATE INDEX IF NOT EXISTS idx_servers_virtfusion_id ON servers("virtfusionId");
CREATE INDEX IF NOT EXISTS idx_servers_server_type ON servers("serverType");
CREATE INDEX IF NOT EXISTS idx_servers_panel_type ON servers("panelType");
CREATE INDEX IF NOT EXISTS idx_servers_owner_id ON servers("ownerId");
CREATE INDEX IF NOT EXISTS idx_servers_node_id ON servers("nodeId");
CREATE INDEX IF NOT EXISTS idx_servers_product_id ON servers("productId");
CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status);
CREATE INDEX IF NOT EXISTS idx_servers_owner_type ON servers("ownerId", "serverType");

-- Server Variables (runtime configuration for servers)
CREATE TABLE IF NOT EXISTS server_variables (
    id TEXT PRIMARY KEY,
    "serverId" TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    "eggVariableId" INTEGER NOT NULL REFERENCES egg_variables(id) ON DELETE CASCADE,
    
    "variableValue" TEXT NOT NULL,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_server_variables_server_id ON server_variables("serverId");
CREATE INDEX IF NOT EXISTS idx_server_variables_egg_variable_id ON server_variables("eggVariableId");

-- Server Properties (flexible key-value store for server specs)
CREATE TABLE IF NOT EXISTS server_properties (
    id TEXT PRIMARY KEY,
    "serverId" TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT server_properties_unique UNIQUE ("serverId", key)
);

CREATE INDEX IF NOT EXISTS idx_server_properties_server_id ON server_properties("serverId");

-- Server Databases (databases associated with servers)
CREATE TABLE IF NOT EXISTS server_databases (
    id TEXT PRIMARY KEY,
    "serverId" TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    host TEXT NOT NULL,
    port INTEGER DEFAULT 3306,
    "databaseName" TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    
    "maxConnections" INTEGER DEFAULT 100,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_server_databases_server_id ON server_databases("serverId");

-- Server Backups (backup records for servers)
CREATE TABLE IF NOT EXISTS server_backups (
    id TEXT PRIMARY KEY,
    "serverId" TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    "fileName" TEXT NOT NULL,
    "fileSize" BIGINT,
    
    "isSuccessful" BOOLEAN DEFAULT true,
    "failureReason" TEXT,
    
    locked BOOLEAN DEFAULT false,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMP,
    "deletedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_server_backups_server_id ON server_backups("serverId");
CREATE INDEX IF NOT EXISTS idx_server_backups_created_at ON server_backups("createdAt");
