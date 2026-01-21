-- schema_10_hytale_audit.sql
-- Hytale audit logging for compliance and security

-- Audit logs for tracking token and session operations
CREATE TABLE IF NOT EXISTS hytale_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Account and profile information
    "accountId" UUID NOT NULL,
    "profileId" UUID,
    
    -- Event type and details
    "eventType" VARCHAR(50) NOT NULL,
    details TEXT, -- JSON details if needed
    
    -- Request context
    "ipAddress" INET,
    "userAgent" TEXT,
    
    -- Timestamps
    "createdAt" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for efficient querying
    FOREIGN KEY ("accountId") REFERENCES hytale_oauth_tokens("accountId") ON DELETE CASCADE
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_hytale_audit_account_id ON hytale_audit_logs("accountId");
CREATE INDEX IF NOT EXISTS idx_hytale_audit_event_type ON hytale_audit_logs("eventType");
CREATE INDEX IF NOT EXISTS idx_hytale_audit_created_at ON hytale_audit_logs("createdAt" DESC);
CREATE INDEX IF NOT EXISTS idx_hytale_audit_account_created ON hytale_audit_logs("accountId", "createdAt" DESC);

-- Add a constraint to validate event types
ALTER TABLE hytale_audit_logs
ADD CONSTRAINT check_valid_event_type CHECK (
    "eventType" IN (
        'TOKEN_CREATED',
        'TOKEN_REFRESHED', 
        'TOKEN_DELETED',
        'AUTH_FAILED',
        'SESSION_CREATED',
        'SESSION_REFRESHED',
        'SESSION_DELETED',
        'PROFILE_SELECTED'
    )
);
