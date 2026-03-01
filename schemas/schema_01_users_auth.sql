-- ============================================================================
-- USERS & AUTHENTICATION SCHEMA
-- ============================================================================

-- Users table for authentication and account management
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password TEXT,
    username TEXT,
    "firstName" TEXT,
    "lastName" TEXT,
    
    roles TEXT[] DEFAULT ARRAY['MEMBER'],
    "isPterodactylAdmin" BOOLEAN DEFAULT false,
    "isVirtfusionAdmin" BOOLEAN DEFAULT false,
    "isSystemAdmin" BOOLEAN DEFAULT false,
    
    "pterodactylId" INTEGER,
    "virtfusionId" INTEGER,
    
    "isMigrated" BOOLEAN DEFAULT false,
    "emailVerified" TIMESTAMP,
    "isActive" BOOLEAN DEFAULT true,
    
    "avatarUrl" TEXT,
    "companyName" TEXT,
    "phoneNumber" TEXT,
    "billingEmail" TEXT,
    
    "accountBalance" DECIMAL(10, 2) DEFAULT 0,
    "accountStatus" TEXT DEFAULT 'active',
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastLoginAt" TIMESTAMP,
    "lastSyncedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_pterodactyl_id ON users("pterodactylId");
CREATE INDEX IF NOT EXISTS idx_users_virtfusion_id ON users("virtfusionId");

-- Sessions for user authentication
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    "sessionToken" TEXT NOT NULL UNIQUE,
    "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires TIMESTAMP NOT NULL,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions("userId");
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires);

-- Password reset tokens
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    "expiresAt" TIMESTAMP NOT NULL,
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens("userId");
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);

-- Verification tokens for email verification, password reset, etc.
CREATE TABLE IF NOT EXISTS verification_tokens (
    identifier TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires TIMESTAMP NOT NULL,
    type TEXT DEFAULT 'email',
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT verification_tokens_unique UNIQUE (identifier, token)
);

CREATE INDEX IF NOT EXISTS idx_verification_tokens_token ON verification_tokens(token);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_identifier ON verification_tokens(identifier);
