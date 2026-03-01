-- Migration: Drop UNIQUE constraint on servers.externalId
-- Pterodactyl does not guarantee that externalId is unique across servers.
-- Multiple servers can have NULL or the same externalId, which violates the
-- constraint during ON CONFLICT ("pterodactylId") upserts because PostgreSQL
-- validates ALL unique constraints before reaching the conflict target.

ALTER TABLE servers DROP CONSTRAINT IF EXISTS "servers_externalId_key";
