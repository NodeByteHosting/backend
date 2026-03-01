-- ============================================================================
-- SUPPORT TICKETS SCHEMA
-- ============================================================================

-- Support Tickets
CREATE TABLE IF NOT EXISTS support_tickets (
    id TEXT PRIMARY KEY,
    "ticketNumber" TEXT NOT NULL UNIQUE,
    "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    "serverId" TEXT REFERENCES servers(id) ON DELETE SET NULL,
    
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    
    status TEXT DEFAULT 'open',
    priority TEXT DEFAULT 'medium',
    category TEXT,
    
    "assignedToId" TEXT REFERENCES users(id) ON DELETE SET NULL,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "closedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_support_tickets_user_id ON support_tickets("userId");
CREATE INDEX IF NOT EXISTS idx_support_tickets_server_id ON support_tickets("serverId");
CREATE INDEX IF NOT EXISTS idx_support_tickets_assigned_to_id ON support_tickets("assignedToId");
CREATE INDEX IF NOT EXISTS idx_support_tickets_status ON support_tickets(status);
CREATE INDEX IF NOT EXISTS idx_support_tickets_priority ON support_tickets(priority);
CREATE INDEX IF NOT EXISTS idx_support_tickets_created_at ON support_tickets("createdAt");

-- Support Ticket Replies
CREATE TABLE IF NOT EXISTS support_ticket_replies (
    id TEXT PRIMARY KEY,
    "ticketId" TEXT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    message TEXT NOT NULL,
    "isInternal" BOOLEAN DEFAULT false,
    
    "createdAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_support_ticket_replies_ticket_id ON support_ticket_replies("ticketId");
CREATE INDEX IF NOT EXISTS idx_support_ticket_replies_user_id ON support_ticket_replies("userId");
CREATE INDEX IF NOT EXISTS idx_support_ticket_replies_created_at ON support_ticket_replies("createdAt");
