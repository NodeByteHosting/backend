-- ============================================================================
-- BILLING & PRODUCTS SCHEMA
-- ============================================================================

-- Products (service/server offerings)
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    
    -- Product type classification for services page filtering
    -- Values: game_server, vps, email, web_hosting, database, cdn, etc.
    "serverType" TEXT NOT NULL DEFAULT 'game_server',
    
    -- Panel integration (specific to game_server type)
    "panelType" TEXT DEFAULT 'pterodactyl',
    
    -- Pterodactyl specific (for game_server type)
    "eggId" INTEGER REFERENCES eggs(id) ON DELETE SET NULL,
    "nestId" INTEGER REFERENCES nests(id) ON DELETE SET NULL,
    
    -- Pricing
    price DECIMAL(10, 2) NOT NULL,
    "billingCycle" TEXT DEFAULT 'monthly',
    "isFree" BOOLEAN DEFAULT false,
    
    -- Flexible specs for different product types
    -- game_server: memory (MB), disk (GB), cpu (cores)
    -- vps: memory (GB), disk (GB), vcpu (cores), bandwidth (Gbps)
    -- email: storage (GB), mailboxes, etc
    "specsMemory" INTEGER,           -- In MB for game servers, GB for VPS
    "specsDisk" INTEGER,             -- In GB
    "specsCpu" DECIMAL(5, 2),        -- In cores
    "specsBandwidth" DECIMAL(5, 2),  -- In Gbps (for VPS/hosting)
    "specsMailboxes" INTEGER,        -- For email hosting
    "specsStorage" INTEGER,          -- In GB (for email/storage)
    
    -- Features stored as JSONB for flexibility across product types
    -- Examples:
    -- game_server: {"autoRestart": true, "dailyBackups": true, "console": true}
    -- vps: {"rootAccess": true, "snapshots": true, "firewalling": true, "ddosProtection": false}
    -- email: {"spamFilter": true, "virusScanning": true, "webmail": true}
    features JSONB DEFAULT '{}',
    
    -- Optional description of what's included
    "includeDescription" TEXT,
    
    -- Metadata
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
CREATE INDEX IF NOT EXISTS idx_products_active_featured ON products("isActive", "isFeatured") WHERE "isActive" = true;

-- Invoices (billing invoices)
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

-- Invoice Items (line items in invoices)
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

-- Payments (payment records)
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
