-- schema_10_hytale_audit.sql
-- Hytale audit logging for compliance and security

-- Audit logs for tracking token and session operations
CREATE TABLE IF NOT EXISTS hytale_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Account and profile information
    account_id UUID NOT NULL,
    profile_id UUID,
    
    -- Event type and details
    event_type VARCHAR(50) NOT NULL,
    details TEXT, -- JSON details if needed
    
    -- Request context
    ip_address INET,
    user_agent TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for efficient querying
    FOREIGN KEY (account_id) REFERENCES hytale_oauth_tokens(account_id) ON DELETE CASCADE
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_hytale_audit_account_id ON hytale_audit_logs(account_id);
CREATE INDEX IF NOT EXISTS idx_hytale_audit_event_type ON hytale_audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_hytale_audit_created_at ON hytale_audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hytale_audit_account_created ON hytale_audit_logs(account_id, created_at DESC);

-- Add a constraint to validate event types
ALTER TABLE hytale_audit_logs
ADD CONSTRAINT check_valid_event_type CHECK (
    event_type IN (
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
