-- ============================================================================
-- SERVERS & RELATED TABLES
-- ============================================================================

-- Servers (game server instances)
CREATE TABLE IF NOT EXISTS servers (
    id TEXT PRIMARY KEY,
    pterodactyl_id INTEGER UNIQUE,
    virtfusion_id INTEGER UNIQUE,
    uuid TEXT NOT NULL UNIQUE,
    uuid_short TEXT,
    external_id TEXT UNIQUE,
    
    panel_type TEXT DEFAULT 'pterodactyl',
    
    name TEXT NOT NULL,
    description TEXT,
    
    status TEXT DEFAULT 'installing',
    is_suspended BOOLEAN DEFAULT false,
    
    product_id TEXT REFERENCES products(id) ON DELETE SET NULL,
    
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    node_id INTEGER NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    egg_id INTEGER REFERENCES eggs(id) ON DELETE SET NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    installed_at TIMESTAMP,
    last_synced_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_servers_uuid ON servers(uuid);
CREATE INDEX IF NOT EXISTS idx_servers_pterodactyl_id ON servers(pterodactyl_id);
CREATE INDEX IF NOT EXISTS idx_servers_virtfusion_id ON servers(virtfusion_id);
CREATE INDEX IF NOT EXISTS idx_servers_panel_type ON servers(panel_type);
CREATE INDEX IF NOT EXISTS idx_servers_owner_id ON servers(owner_id);
CREATE INDEX IF NOT EXISTS idx_servers_node_id ON servers(node_id);
CREATE INDEX IF NOT EXISTS idx_servers_product_id ON servers(product_id);

-- Server Variables (runtime configuration for servers)
CREATE TABLE IF NOT EXISTS server_variables (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    egg_variable_id INTEGER NOT NULL REFERENCES egg_variables(id) ON DELETE CASCADE,
    
    variable_value TEXT NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_server_variables_server_id ON server_variables(server_id);
CREATE INDEX IF NOT EXISTS idx_server_variables_egg_variable_id ON server_variables(egg_variable_id);

-- Server Properties (flexible key-value store for server specs)
CREATE TABLE IF NOT EXISTS server_properties (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT server_properties_unique UNIQUE (server_id, key)
);

CREATE INDEX IF NOT EXISTS idx_server_properties_server_id ON server_properties(server_id);

-- Server Databases (databases associated with servers)
CREATE TABLE IF NOT EXISTS server_databases (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    host TEXT NOT NULL,
    port INTEGER DEFAULT 3306,
    database_name TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    
    max_connections INTEGER DEFAULT 100,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_server_databases_server_id ON server_databases(server_id);

-- Server Backups (backup records for servers)
CREATE TABLE IF NOT EXISTS server_backups (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    
    file_name TEXT NOT NULL,
    file_size BIGINT,
    
    is_successful BOOLEAN DEFAULT true,
    failure_reason TEXT,
    
    locked BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_server_backups_server_id ON server_backups(server_id);
CREATE INDEX IF NOT EXISTS idx_server_backups_created_at ON server_backups(created_at);
