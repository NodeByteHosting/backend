-- Migration: create all missing tables in dependency-safe order
-- Uses CREATE TABLE IF NOT EXISTS throughout — safe to re-run

-- ============================================================================
-- SCHEMA 02: locations → nodes → nests → eggs → egg_variables/properties
-- (allocations comes AFTER servers due to circular FK)
-- ============================================================================

CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY,
    "shortCode" TEXT NOT NULL UNIQUE,
    description TEXT,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_locations_short_code ON locations("shortCode");

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
CREATE INDEX IF NOT EXISTS idx_eggs_nest_id ON eggs("nestId");
CREATE INDEX IF NOT EXISTS idx_eggs_panel_type ON eggs("panelType");

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

-- ============================================================================
-- SCHEMA 04 (partial): products (depends on eggs/nests/users)
-- ============================================================================

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    "serverType" TEXT NOT NULL DEFAULT 'game_server',
    "panelType" TEXT DEFAULT 'pterodactyl',
    "eggId" INTEGER REFERENCES eggs(id) ON DELETE SET NULL,
    "nestId" INTEGER REFERENCES nests(id) ON DELETE SET NULL,
    price DECIMAL(10, 2) NOT NULL,
    "billingCycle" TEXT DEFAULT 'monthly',
    "isFree" BOOLEAN DEFAULT false,
    "specsMemory" INTEGER,
    "specsDisk" INTEGER,
    "specsCpu" DECIMAL(5, 2),
    "specsBandwidth" DECIMAL(5, 2),
    "specsMailboxes" INTEGER,
    "specsStorage" INTEGER,
    features JSONB DEFAULT '{}',
    "includeDescription" TEXT,
    "createdById" TEXT REFERENCES users(id) ON DELETE SET NULL,
    "isActive" BOOLEAN DEFAULT true,
    "isFeatured" BOOLEAN DEFAULT false,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_products_slug ON products(slug);
CREATE INDEX IF NOT EXISTS idx_products_server_type ON products("serverType");
CREATE INDEX IF NOT EXISTS idx_products_egg_id ON products("eggId");
CREATE INDEX IF NOT EXISTS idx_products_created_by_id ON products("createdById");
CREATE INDEX IF NOT EXISTS idx_products_is_active ON products("isActive");

-- ============================================================================
-- SCHEMA 03: servers (depends on eggs/nests/nodes/products/users)
-- ============================================================================

CREATE TABLE IF NOT EXISTS servers (
    id TEXT PRIMARY KEY,
    "serverType" TEXT NOT NULL DEFAULT 'game_server',
    "pterodactylId" INTEGER UNIQUE,
    "virtfusionId" INTEGER UNIQUE,
    uuid TEXT UNIQUE,
    "uuidShort" TEXT,
    "externalId" TEXT UNIQUE,
    "panelType" TEXT DEFAULT 'pterodactyl',
    "eggId" INTEGER REFERENCES eggs(id) ON DELETE SET NULL,
    "nestId" INTEGER REFERENCES nests(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'installing',
    "isSuspended" BOOLEAN DEFAULT false,
    "productId" TEXT REFERENCES products(id) ON DELETE SET NULL,
    "ownerId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "nodeId" INTEGER REFERENCES nodes(id) ON DELETE SET NULL,
    config JSONB DEFAULT '{}',
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

-- Now allocations can reference servers
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

-- Remaining server tables
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

-- ============================================================================
-- SCHEMA 04 (remaining): invoices, invoice_items, payments
-- ============================================================================

CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    "invoiceNumber" TEXT NOT NULL UNIQUE,
    "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL,
    tax DECIMAL(10, 2) DEFAULT 0,
    total DECIMAL(10, 2) NOT NULL,
    status TEXT DEFAULT 'unpaid',
    "paymentMethod" TEXT,
    "paidAt" TIMESTAMP,
    "dueAt" TIMESTAMP,
    notes TEXT,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_invoices_user_id ON invoices("userId");
CREATE INDEX IF NOT EXISTS idx_invoices_invoice_number ON invoices("invoiceNumber");
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_created_at ON invoices("createdAt");

CREATE TABLE IF NOT EXISTS invoice_items (
    id TEXT PRIMARY KEY,
    "invoiceId" TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity INTEGER DEFAULT 1,
    "unitPrice" DECIMAL(10, 2) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    "productId" TEXT REFERENCES products(id) ON DELETE SET NULL,
    "serverId" TEXT REFERENCES servers(id) ON DELETE SET NULL,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id ON invoice_items("invoiceId");
CREATE INDEX IF NOT EXISTS idx_invoice_items_product_id ON invoice_items("productId");
CREATE INDEX IF NOT EXISTS idx_invoice_items_server_id ON invoice_items("serverId");

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    "invoiceId" TEXT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL,
    "paymentMethod" TEXT NOT NULL,
    "externalTransactionId" TEXT UNIQUE,
    status TEXT DEFAULT 'completed',
    notes TEXT,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id ON payments("invoiceId");
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments("userId");
CREATE INDEX IF NOT EXISTS idx_payments_external_transaction_id ON payments("externalTransactionId");
