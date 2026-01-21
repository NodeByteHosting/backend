-- ============================================================================
-- PARTNERS SCHEMA - Integration & Partnership Management
-- ============================================================================

-- Partners (hosting providers, integrations, collaborators)
CREATE TABLE IF NOT EXISTS partners (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    
    -- Partner classification
    -- Values: hosting_provider, integration, reseller, affiliate, technology_partner
    "partnerType" TEXT NOT NULL,
    
    -- Contact and branding
    website TEXT,
    logo_url TEXT,
    "contactEmail" TEXT,
    "contactPerson" TEXT,
    
    -- Partnership details
    "partnershipStartDate" TIMESTAMP,
    "partnershipEndDate" TIMESTAMP,
    status TEXT DEFAULT 'active', -- active, inactive, pending, suspended
    
    -- Metadata
    metadata JSONB DEFAULT '{}', -- API keys, endpoints, terms, etc.
    
    -- Admin controls
    "createdById" TEXT REFERENCES users(id) ON DELETE SET NULL,
    
    "isActive" BOOLEAN DEFAULT true,
    "isFeatured" BOOLEAN DEFAULT false,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_partners_slug ON partners(slug);
CREATE INDEX IF NOT EXISTS idx_partners_type ON partners("partnerType");
CREATE INDEX IF NOT EXISTS idx_partners_status ON partners(status);
CREATE INDEX IF NOT EXISTS idx_partners_is_active ON partners("isActive");
CREATE INDEX IF NOT EXISTS idx_partners_is_featured ON partners("isFeatured");

-- Partner Services (what services each partner provides/integrates)
CREATE TABLE IF NOT EXISTS partner_services (
    id TEXT PRIMARY KEY,
    "partnerId" TEXT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    
    name TEXT NOT NULL,
    description TEXT,
    
    -- Service configuration/endpoints
    config JSONB DEFAULT '{}',
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_partner_services_partner_id ON partner_services("partnerId");

-- Partner Revenue Sharing (commission structure and tracking)
CREATE TABLE IF NOT EXISTS partner_revenue_sharing (
    id TEXT PRIMARY KEY,
    "partnerId" TEXT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    
    -- Commission structure
    "commissionType" TEXT NOT NULL, -- percentage, fixed, tiered
    "commissionRate" DECIMAL(5, 2), -- for percentage: 0-100
    "commissionAmount" DECIMAL(10, 2), -- for fixed: amount
    
    -- Payout details
    "payoutFrequency" TEXT DEFAULT 'monthly', -- monthly, quarterly, yearly
    "minimumPayout" DECIMAL(10, 2),
    "payoutMethod" TEXT, -- bank_transfer, paypal, crypto, etc.
    "payoutAccount" TEXT, -- encrypted account details
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_partner_revenue_sharing_partner_id ON partner_revenue_sharing("partnerId");
