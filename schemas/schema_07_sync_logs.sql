-- ============================================================================
-- SYNC LOGS SCHEMA
-- ============================================================================

-- Sync Logs (track synchronization history from panels)
CREATE TABLE IF NOT EXISTS sync_logs (
    id TEXT PRIMARY KEY,
    sync_type TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    
    records_synced INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,
    records_total INTEGER DEFAULT 0,
    
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_seconds INTEGER,
    
    error_message TEXT,
    metadata JSONB,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_logs_sync_type ON sync_logs(sync_type);
CREATE INDEX IF NOT EXISTS idx_sync_logs_status ON sync_logs(status);
CREATE INDEX IF NOT EXISTS idx_sync_logs_created_at ON sync_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_logs_started_at ON sync_logs(started_at);
