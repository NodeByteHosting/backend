-- ============================================================================
-- SYNC LOGS SCHEMA
-- ============================================================================

-- Sync Logs (track synchronization history from panels)
CREATE TABLE IF NOT EXISTS sync_logs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT DEFAULT 'pending',

    "itemsTotal" INTEGER DEFAULT 0,
    "itemsSynced" INTEGER DEFAULT 0,
    "itemsFailed" INTEGER DEFAULT 0,

    "startedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMP,
    "durationSeconds" INTEGER,

    error TEXT,
    metadata JSONB,

    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_logs_type ON sync_logs(type);
CREATE INDEX IF NOT EXISTS idx_sync_logs_status ON sync_logs(status);
CREATE INDEX IF NOT EXISTS idx_sync_logs_created_at ON sync_logs("createdAt");
CREATE INDEX IF NOT EXISTS idx_sync_logs_started_at ON sync_logs("startedAt");
