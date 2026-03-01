-- Migration: Add resource limit columns to servers table
-- These columns store Pterodactyl panel limits synced during server sync.

ALTER TABLE servers
  ADD COLUMN IF NOT EXISTS memory INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS disk   INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS cpu    INTEGER NOT NULL DEFAULT 0;
