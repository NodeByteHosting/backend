-- ============================================================================
-- PTERODACTYL SYNC MODELS - Panel infrastructure data
-- ============================================================================

-- Locations (data center regions)
CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY,
    "shortCode" TEXT NOT NULL UNIQUE,
    description TEXT,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_locations_short_code ON locations("shortCode");

-- Nodes (physical/virtual servers hosting game servers)
CREATE TABLE IF NOT EXISTS nodes (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    fqdn TEXT NOT NULL,
    scheme TEXT DEFAULT 'https',
    "behindProxy" BOOLEAN DEFAULT false,
    
    "panelType" TEXT DEFAULT 'pterodactyl',
    
    memory BIGINT NOT NULL,
    "memoryOverallocate" INTEGER DEFAULT 0,
    disk BIGINT NOT NULL,
    "diskOverallocate" INTEGER DEFAULT 0,
    
    "isPublic" BOOLEAN DEFAULT true,
    "isMaintenanceMode" BOOLEAN DEFAULT false,
    
    "daemonListenPort" INTEGER DEFAULT 8080,
    "daemonSftpPort" INTEGER DEFAULT 2022,
    "daemonBase" TEXT DEFAULT '/var/lib/pterodactyl/volumes',
    
    "locationId" INTEGER NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nodes_uuid ON nodes(uuid);
CREATE INDEX IF NOT EXISTS idx_nodes_panel_type ON nodes("panelType");
CREATE INDEX IF NOT EXISTS idx_nodes_location_id ON nodes("locationId");

-- Allocations (IP:Port combinations on nodes)
CREATE TABLE IF NOT EXISTS allocations (
    id INTEGER PRIMARY KEY,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    alias TEXT,
    notes TEXT,
    "isAssigned" BOOLEAN DEFAULT false,
    
    "nodeId" INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    "serverId" TEXT REFERENCES servers(id) ON DELETE SET NULL,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT allocations_unique UNIQUE (ip, port)
);

CREATE INDEX IF NOT EXISTS idx_allocations_node_id ON allocations("nodeId");
CREATE INDEX IF NOT EXISTS idx_allocations_server_id ON allocations("serverId");
CREATE INDEX IF NOT EXISTS idx_allocations_ip_port ON allocations(ip, port);

-- Nests (server type categories like Minecraft, Rust)
CREATE TABLE IF NOT EXISTS nests (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    author TEXT,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nests_uuid ON nests(uuid);

-- Eggs (server type templates like Paper, Vanilla, Forge)
CREATE TABLE IF NOT EXISTS eggs (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    author TEXT,
    
    "panelType" TEXT DEFAULT 'pterodactyl',
    
    "nestId" INTEGER NOT NULL REFERENCES nests(id) ON DELETE CASCADE,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_eggs_uuid ON eggs(uuid);
CREATE INDEX IF NOT EXISTS idx_eggs_nest_id ON eggs(nest_id);
CREATE INDEX IF NOT EXISTS idx_eggs_panel_type ON eggs(panel_type);

-- Egg Variables (configuration options for eggs)
CREATE TABLE IF NOT EXISTS egg_variables (
    id INTEGER PRIMARY KEY,
    egg_id INTEGER NOT NULL REFERENCES eggs(id) ON DELETE CASCADE,
    
    name TEXT NOT NULL,
    description TEXT,
    env_variable TEXT NOT NULL,
    default_value TEXT,
    user_viewable BOOLEAN DEFAULT true,
    user_editable BOOLEAN DEFAULT true,
    rules TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_egg_variables_egg_id ON egg_variables(egg_id);

-- Egg Properties (flexible key-value store for egg-specific config)
CREATE TABLE IF NOT EXISTS egg_properties (
    id TEXT PRIMARY KEY,
    egg_id INTEGER NOT NULL REFERENCES eggs(id) ON DELETE CASCADE,
    
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    panel_type TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT egg_properties_unique UNIQUE (egg_id, key, panel_type)
);

CREATE INDEX IF NOT EXISTS idx_egg_properties_egg_id ON egg_properties(egg_id);
CREATE INDEX IF NOT EXISTS idx_egg_properties_key ON egg_properties(key);
