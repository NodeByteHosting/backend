-- Make ownerId nullable on servers so servers can be synced before their owner user exists locally.
-- The users sync runs after servers, so the first sync would always fail with NOT NULL violation.
-- ownerId is reconciled during users sync via COALESCE logic in the upsert.

ALTER TABLE servers ALTER COLUMN "ownerId" DROP NOT NULL;
