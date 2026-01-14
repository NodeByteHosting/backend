# NodeByte Database Schemas

Complete SQL schema files for setting up a NodeByte database from scratch without Prisma.

## Overview

This directory contains modular SQL schema files that define the complete NodeByte database structure:

| File | Tables | Purpose |
|------|--------|---------|
| `schema_01_users_auth.sql` | users, sessions, password_reset_tokens, verification_tokens | User authentication and account management |
| `schema_02_pterodactyl_sync.sql` | locations, nodes, allocations, nests, eggs, egg_variables, egg_properties | Game panel infrastructure sync data |
| `schema_03_servers.sql` | servers, server_variables, server_properties, server_databases, server_backups | Game server instances and configuration |
| `schema_04_billing.sql` | products, invoices, invoice_items, payments | Billing and commerce system |
| `schema_05_support_tickets.sql` | support_tickets, support_ticket_replies | Customer support ticketing |
| `schema_06_discord_webhooks.sql` | discord_webhooks | Discord webhook management for notifications |
| `schema_07_sync_logs.sql` | sync_logs | Synchronization history from panels |
| `schema_08_config.sql` | config | System configuration key-value store |
| `schema_hytale.sql` | hytale_oauth_tokens, hytale_game_sessions | Hytale OAuth tokens and game sessions |

## Quick Start

### Linux / macOS

```bash
cd backend/schemas
chmod +x init-database.sh
./init-database.sh "postgresql://user:password@localhost:5432/nodebyte"
```

### Windows

```cmd
cd backend\schemas
init-database.bat "postgresql://user:password@localhost:5432/nodebyte"
```

### Manual Setup

If you prefer to execute schemas manually:

```bash
psql postgresql://user:password@localhost:5432/nodebyte -f schema_01_users_auth.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_02_pterodactyl_sync.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_03_servers.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_04_billing.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_05_support_tickets.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_06_discord_webhooks.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_07_sync_logs.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_08_config.sql
psql postgresql://user:password@localhost:5432/nodebyte -f schema_hytale.sql
```

## Schema Details

### Users & Authentication

**Tables:**
- `users` - User accounts with roles, panel IDs, and billing info
- `sessions` - Active user sessions with JWT tokens
- `password_reset_tokens` - Password reset token management
- `verification_tokens` - Email verification and similar tokens

**Key Features:**
- Support for multiple panel types (Pterodactyl, Virtfusion)
- Admin role system (Pterodactyl admin, Virtfusion admin, System admin)
- Account balance tracking
- Account status management

### Pterodactyl Sync

**Tables:**
- `locations` - Data center regions
- `nodes` - Physical/virtual servers
- `allocations` - IP:Port combinations on nodes
- `nests` - Server type categories (Minecraft, Rust, etc.)
- `eggs` - Server type templates (Paper, Vanilla, Forge, etc.)
- `egg_variables` - Configuration options for eggs
- `egg_properties` - Flexible key-value store for egg-specific config

**Key Features:**
- Mirrors Pterodactyl panel data structure
- Support for multiple panel types
- Resource allocation tracking (memory, disk, CPU)
- Flexible properties for panel-specific configuration

### Servers

**Tables:**
- `servers` - Game server instances
- `server_variables` - Runtime configuration values
- `server_properties` - Flexible key-value specs and config
- `server_databases` - Database credentials and connections
- `server_backups` - Backup records and history

**Key Features:**
- Supports both Pterodactyl and Virtfusion panels
- Server status tracking (installing, running, suspended)
- Resource specifications (memory, disk, CPU)
- Product/billing linkage
- Backup management

### Billing

**Tables:**
- `products` - Service offerings with pricing
- `invoices` - Customer invoices
- `invoice_items` - Line items in invoices
- `payments` - Payment records

**Key Features:**
- Support for multiple billing cycles (monthly, yearly, etc.)
- Invoice status tracking (paid, unpaid, overdue)
- Product associations with server types
- External payment provider integration support

### Support Tickets

**Tables:**
- `support_tickets` - Customer support tickets
- `support_ticket_replies` - Ticket responses and comments

**Key Features:**
- Ticket status tracking (open, in-progress, closed)
- Priority levels (low, medium, high)
- Assignment to support staff
- Internal/external note separation

### Discord Webhooks

**Table:**
- `discord_webhooks` - Discord webhook configurations

**Key Features:**
- Per-server or account-wide webhooks
- Granular notification options (server events, player events, backups, etc.)
- Webhook validation and status tracking
- Custom message templates

### Sync Logs

**Table:**
- `sync_logs` - Synchronization operation history

**Key Features:**
- Sync type tracking (full, locations, nodes, servers, users)
- Status monitoring (pending, in-progress, completed, failed)
- Record counts (synced, failed, total)
- Duration tracking
- Error message logging
- JSON metadata for debugging

### Config

**Table:**
- `config` - System configuration key-value store

**Key Features:**
- Simple key-value storage for system settings
- Privacy controls (public/private)
- Timestamped updates

### Hytale OAuth

**Tables:**
- `hytale_oauth_tokens` - Hytale OAuth tokens
- `hytale_game_sessions` - Active game sessions for Hytale servers

**Key Features:**
- OAuth token persistence with expiry tracking
- Refresh token management
- Game profile association
- Session token and identity token storage
- Automatic expiry management

## Database Requirements

- **PostgreSQL 12+** (uses UUID, JSONB, and other modern features)
- **Minimum 100MB** initial disk space
- **UTF-8 encoding** support

## Environment Variables

When running the backend, configure these:

```env
DATABASE_URL=postgresql://user:password@localhost:5432/nodebyte
```

## Schema Relationships

```
users (1) ──→ (many) sessions
users (1) ──→ (many) servers
users (1) ──→ (many) invoices
users (1) ──→ (many) support_tickets
users (1) ──→ (many) discord_webhooks

servers (many) ──→ (1) users (owner)
servers (many) ──→ (1) nodes
servers (many) ──→ (1) eggs
servers (many) ──→ (1) products

nodes (many) ──→ (1) locations
allocations (many) ──→ (1) nodes
allocations (many) ──→ (1) servers

eggs (many) ──→ (1) nests
egg_variables (many) ──→ (1) eggs
egg_properties (many) ──→ (1) eggs
server_variables (many) ──→ (1) egg_variables

invoices (many) ──→ (1) users
invoice_items (many) ──→ (1) invoices
payments (many) ──→ (1) invoices

support_tickets (many) ──→ (1) users (creator)
support_ticket_replies (many) ──→ (1) support_tickets
```

## Indexing Strategy

All schemas include strategic indexes for:
- Foreign key lookups
- Filtering by status/type
- Sorting by timestamps
- Unique constraint enforcement
- High-cardinality fields

## Migration from Prisma

If migrating from a Prisma-managed database:

1. Export data from existing Prisma schema
2. Run these schema files on new database
3. Import data using `psql COPY` or similar tools
4. Verify data integrity

## Troubleshooting

### Connection Errors

```bash
# Test connection
psql postgresql://user:password@localhost:5432/nodebyte -c "SELECT 1"
```

### Permission Errors

Ensure your database user has:
- `CREATE TABLE` permission
- `CREATE INDEX` permission
- Ownership of the target database

```sql
-- Grant permissions
ALTER ROLE your_user CREATEDB CREATEROLE;
```

### Schema Already Exists

These scripts use `IF NOT EXISTS` clauses, so re-running is safe. Tables will not be recreated.

## Performance Tuning

For production databases, consider:

```sql
-- Analyze tables for query planning
ANALYZE;

-- Rebuild indexes
REINDEX DATABASE nodebyte;
```

## Backup & Restore

### Backup

```bash
pg_dump postgresql://user:password@localhost:5432/nodebyte > nodebyte_backup.sql
```

### Restore

```bash
psql postgresql://user:password@localhost:5432/nodebyte < nodebyte_backup.sql
```

## Support

For issues with schema setup, check:
- PostgreSQL logs
- Database user permissions
- Network connectivity to database server

For NodeByte-specific questions, see the main README.md
