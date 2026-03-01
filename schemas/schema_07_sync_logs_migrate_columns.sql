-- Migration: rename sync_logs columns to match Go code expectations
-- Safe to run multiple times (checks column existence before renaming)

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sync_logs' AND column_name='syncType') THEN
        ALTER TABLE sync_logs RENAME COLUMN "syncType" TO type;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sync_logs' AND column_name='recordsTotal') THEN
        ALTER TABLE sync_logs RENAME COLUMN "recordsTotal" TO "itemsTotal";
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sync_logs' AND column_name='recordsSynced') THEN
        ALTER TABLE sync_logs RENAME COLUMN "recordsSynced" TO "itemsSynced";
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sync_logs' AND column_name='recordsFailed') THEN
        ALTER TABLE sync_logs RENAME COLUMN "recordsFailed" TO "itemsFailed";
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='sync_logs' AND column_name='errorMessage') THEN
        ALTER TABLE sync_logs RENAME COLUMN "errorMessage" TO error;
    END IF;
END $$;

-- Recreate index on renamed column
DROP INDEX IF EXISTS idx_sync_logs_sync_type;
CREATE INDEX IF NOT EXISTS idx_sync_logs_type ON sync_logs(type);
