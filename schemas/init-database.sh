#!/bin/bash
# ============================================================================
# NodeByte Database Initialization Script
# ============================================================================
# This script initializes the complete NodeByte database schema
# Usage: ./init-database.sh <DATABASE_URL>
#
# Example:
#   ./init-database.sh "postgresql://user:password@localhost:5432/nodebyte"
#
# ============================================================================

set -e

# Check if DATABASE_URL is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <DATABASE_URL>"
    echo ""
    echo "Example:"
    echo "  $0 'postgresql://user:password@localhost:5432/nodebyte'"
    echo ""
    exit 1
fi

DATABASE_URL="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "============================================================================"
echo "NodeByte Database Initialization"
echo "============================================================================"
echo ""
echo "Database URL: $DATABASE_URL (password hidden)"
echo "Schema directory: $SCRIPT_DIR"
echo ""

# Array of schema files in execution order
SCHEMAS=(
    "schema_01_users_auth.sql"
    "schema_02_pterodactyl_sync.sql"
    "schema_03_servers.sql"
    "schema_04_billing.sql"
    "schema_05_support_tickets.sql"
    "schema_06_discord_webhooks.sql"
    "schema_07_sync_logs.sql"
    "schema_08_config.sql"
    "schema_09_hytale.sql"
)

# Execute each schema file
for schema in "${SCHEMAS[@]}"; do
    schema_path="$SCRIPT_DIR/$schema"
    
    if [ ! -f "$schema_path" ]; then
        echo "❌ Schema file not found: $schema_path"
        exit 1
    fi
    
    echo "📦 Executing: $schema"
    psql "$DATABASE_URL" -f "$schema_path" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo "✅ $schema"
    else
        echo "❌ Failed to execute $schema"
        exit 1
    fi
done

echo ""
echo "============================================================================"
echo "✅ Database initialization complete!"
echo "============================================================================"
echo ""
echo "Summary:"
echo "  - Users & Authentication (users, sessions, password_reset_tokens, verification_tokens)"
echo "  - Pterodactyl Sync (locations, nodes, allocations, nests, eggs, egg_variables, egg_properties)"
echo "  - Servers (servers, server_variables, server_properties, server_databases, server_backups)"
echo "  - Billing (products, invoices, invoice_items, payments)"
echo "  - Support (support_tickets, support_ticket_replies)"
echo "  - Discord Webhooks (discord_webhooks)"
echo "  - Sync Logs (sync_logs)"
echo "  - Config (config)"
echo "  - Hytale OAuth (hytale_oauth_tokens, hytale_game_sessions)"
echo ""
echo "You can now start your backend with:"
echo "  go run ./cmd/api/main.go"
echo ""
