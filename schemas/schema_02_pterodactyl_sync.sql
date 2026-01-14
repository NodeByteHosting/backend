-- ============================================================================
-- PTERODACTYL SYNC MODELS - Panel infrastructure data
-- ============================================================================

-- Locations (data center regions)
CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY,
    short_code TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_locations_short_code ON locations(short_code);

-- Nodes (physical/virtual servers hosting game servers)
CREATE TABLE IF NOT EXISTS nodes (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    fqdn TEXT NOT NULL,
    scheme TEXT DEFAULT 'https',
    behind_proxy BOOLEAN DEFAULT false,
    
    panel_type TEXT DEFAULT 'pterodactyl',
    
    memory BIGINT NOT NULL,
    memory_overallocate INTEGER DEFAULT 0,
    disk BIGINT NOT NULL,
    disk_overallocate INTEGER DEFAULT 0,
    
    is_public BOOLEAN DEFAULT true,
    is_maintenance_mode BOOLEAN DEFAULT false,
    
    daemon_listen_port INTEGER DEFAULT 8080,
    daemon_sftp_port INTEGER DEFAULT 2022,
    daemon_base TEXT DEFAULT '/var/lib/pterodactyl/volumes',
    
    location_id INTEGER NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nodes_uuid ON nodes(uuid);
CREATE INDEX IF NOT EXISTS idx_nodes_panel_type ON nodes(panel_type);
CREATE INDEX IF NOT EXISTS idx_nodes_location_id ON nodes(location_id);

-- Allocations (IP:Port combinations on nodes)
CREATE TABLE IF NOT EXISTS allocations (
    id INTEGER PRIMARY KEY,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    alias TEXT,
    notes TEXT,
    is_assigned BOOLEAN DEFAULT false,
    
    node_id INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    server_id TEXT REFERENCES servers(id) ON DELETE SET NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT allocations_unique UNIQUE (ip, port)
);

CREATE INDEX IF NOT EXISTS idx_allocations_node_id ON allocations(node_id);
CREATE INDEX IF NOT EXISTS idx_allocations_server_id ON allocations(server_id);
CREATE INDEX IF NOT EXISTS idx_allocations_ip_port ON allocations(ip, port);

-- Nests (server type categories like Minecraft, Rust)
CREATE TABLE IF NOT EXISTS nests (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    author TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_nests_uuid ON nests(uuid);

-- Eggs (server type templates like Paper, Vanilla, Forge)
CREATE TABLE IF NOT EXISTS eggs (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    author TEXT,
    
    panel_type TEXT DEFAULT 'pterodactyl',
    
    nest_id INTEGER NOT NULL REFERENCES nests(id) ON DELETE CASCADE,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
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
