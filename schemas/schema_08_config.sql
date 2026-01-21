-- ============================================================================
-- CONFIG & SETTINGS SCHEMA
-- ============================================================================

-- System Configuration (key-value store for system settings)
CREATE TABLE IF NOT EXISTS config (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    description TEXT,
    "isPublic" BOOLEAN DEFAULT false,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_config_key ON config(key);
