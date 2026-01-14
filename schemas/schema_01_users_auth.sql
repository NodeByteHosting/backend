-- ============================================================================
-- USERS & AUTHENTICATION SCHEMA
-- ============================================================================

-- Users table for authentication and account management
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password TEXT,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    
    roles TEXT[] DEFAULT ARRAY['MEMBER'],
    is_pterodactyl_admin BOOLEAN DEFAULT false,
    is_virtfusion_admin BOOLEAN DEFAULT false,
    is_system_admin BOOLEAN DEFAULT false,
    
    pterodactyl_id INTEGER,
    virtfusion_id INTEGER,
    
    is_migrated BOOLEAN DEFAULT false,
    email_verified TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    
    avatar_url TEXT,
    company_name TEXT,
    phone_number TEXT,
    billing_email TEXT,
    
    account_balance DECIMAL(10, 2) DEFAULT 0,
    account_status TEXT DEFAULT 'active',
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    last_synced_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_pterodactyl_id ON users(pterodactyl_id);
CREATE INDEX IF NOT EXISTS idx_users_virtfusion_id ON users(virtfusion_id);

-- Sessions for user authentication
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    session_token TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires);

-- Password reset tokens
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);

-- Verification tokens for email verification, password reset, etc.
CREATE TABLE IF NOT EXISTS verification_tokens (
    identifier TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires TIMESTAMP NOT NULL,
    type TEXT DEFAULT 'email',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT verification_tokens_unique UNIQUE (identifier, token)
);

CREATE INDEX IF NOT EXISTS idx_verification_tokens_token ON verification_tokens(token);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_identifier ON verification_tokens(identifier);
