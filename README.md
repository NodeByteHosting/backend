# NodeByte Backend API

REST API and background job processing service for NodeByte infrastructure management.

[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-AGPL%203.0-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-brightgreen.svg)](Dockerfile)
[![Tests](https://github.com/NodeByteHosting/nodebyte-host/actions/workflows/test-build.yml/badge.svg)](https://github.com/NodeByteHosting/nodebyte-host/actions)
[![Lint](https://github.com/NodeByteHosting/nodebyte-host/actions/workflows/lint.yml/badge.svg)](https://github.com/NodeByteHosting/nodebyte-host/actions)
[![Coverage](https://codecov.io/gh/NodeByteHosting/nodebyte-host/branch/master/graph/badge.svg)](https://codecov.io/gh/NodeByteHosting/nodebyte-host)

## Overview

The NodeByte Backend API provides a comprehensive REST API for managing game server infrastructure, with enterprise-grade features:

- **Hytale OAuth 2.0 Authentication** - Device code flow, token management, game session handling with JWT validation
- **Panel Synchronization** - Full Pterodactyl panel sync (locations, nodes, allocations, nests, eggs, servers, users, databases)
- **Job Queue System** - Redis-backed async job processing with priority queues (Asynq)
- **Admin Dashboard** - Complete REST API for system settings, webhooks, and sync management
- **Email Queue** - Asynchronous email sending via Resend API
- **Discord Webhooks** - Real-time notifications for sync events and system changes
- **Cron Scheduler** - Automated scheduled sync jobs with configurable intervals
- **Rate Limiting** - Token bucket algorithm with per-endpoint configuration
- **Audit Logging** - Compliance-grade event tracking with database persistence
- **Database Configuration** - Dynamic system settings with AES-256-GCM encryption
- **Health Monitoring** - Built-in health checks and comprehensive structured logging

## Quick Start

### Prerequisites
- Go 1.24+
- PostgreSQL 12+
- Redis 6+
- Docker & Docker Compose (optional)

### Local Development

```bash
# Clone repository
cd backend

# Install dependencies
go mod download
go mod tidy

# Create environment file
cp .env.example .env

# Edit configuration (set PTERODACTYL_URL, DATABASE_URL, etc.)
nano .env

# Start dependencies (Redis + PostgreSQL)
docker-compose up -d postgres redis

# Run database migrations
# (Run these from the main app or apply schemas manually)

# Run server
go run ./cmd/api/main.go

# Server runs at http://localhost:8080
# Health check: curl http://localhost:8080/health
```

### Docker Deployment

```bash
# Start all services (backend + postgres + redis)
docker-compose up -d

# With Asynq web UI for job monitoring
docker-compose --profile monitoring up -d

# View live logs
docker-compose logs -f backend

# Shutdown
docker-compose down
```

### Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# Get statistics
curl http://localhost:8080/api/stats

# Trigger sync (requires API key)
curl -X POST http://localhost:8080/api/v1/sync/full \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"skip_users": false}'
```

## Configuration

### Environment Variables

```bash
# Server
ENV=production                          # development or production
BACKEND_PORT=8080                       # HTTP server port

# Database
DATABASE_URL=postgresql://user:pass@localhost/nodebyte  # Required

# Redis
REDIS_URL=redis://localhost:6379        # Can also be: host:port

# Security
BACKEND_API_KEY=your-secret-api-key     # For X-API-Key authentication
CORS_ORIGINS=https://app.example.com    # Comma-separated origins
ENCRYPTION_KEY=32-byte-hex-encoded-key  # For encrypting sensitive values

# Pterodactyl Panel
PTERODACTYL_URL=https://panel.example.com          # Required
PTERODACTYL_API_KEY=your-admin-api-key             # Required
PTERODACTYL_CLIENT_API_KEY=client-api-key          # Optional

# Hytale OAuth (Required for game server authentication)
HYTALE_USE_STAGING=false                # false for production, true for staging Hytale OAuth
# Tokens auto-refresh every 5-10 minutes

# Virtfusion Panel (optional)
VIRTFUSION_URL=https://virtfusion.example.com
VIRTFUSION_API_KEY=your-api-key

# Email (Resend)
RESEND_API_KEY=re_xxxxxxxxxxxxx        # Required for email sending
EMAIL_FROM=noreply@example.com

# Sync Settings
AUTO_SYNC_ENABLED=true                  # Enable scheduled syncs
AUTO_SYNC_INTERVAL=3600                 # Interval in seconds (1 hour)
SYNC_BATCH_SIZE=100                     # Items per batch during sync

# Scalar (optional)
SCALAR_URL=https://scalar.example.com
SCALAR_API_KEY=your-api-key

# Cloudflare (optional)
CF_ACCESS_CLIENT_ID=your-client-id
CF_ACCESS_CLIENT_SECRET=your-client-secret
```

### Database Setup

The backend uses the same PostgreSQL database as the main application. Required tables are created automatically via migrations. Key tables include:
- `users` / `accounts` - User authentication
- `hytale_oauth_tokens` - OAuth token storage (encrypted)
- `hytale_audit_logs` - Compliance audit trail
- `sync_logs` - Pterodactyl sync operation logs
- `webhooks` - Discord webhook configurations

## Hytale OAuth 2.0 Integration

The backend implements complete OAuth 2.0 Device Code Flow for Hytale server authentication, enabling secure game session management.

### Key Features

✅ **Device Code Flow** - RFC 8628 compliant device authorization  
✅ **Token Management** - Automatic refresh with 30-day validity  
✅ **JWT Validation** - Ed25519 signature verification with JWKS caching  
✅ **Game Sessions** - Per-player session tokens with 1-hour auto-refresh  
✅ **Audit Logging** - Complete compliance trail of all OAuth operations  
✅ **Rate Limiting** - Token bucket algorithm (5-20 requests per minute)  
✅ **Error Handling** - Graceful session limit & expiration handling  

### OAuth Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/hytale/oauth/device-code` | POST | Request device code for browser auth |
| `/api/v1/hytale/oauth/token` | POST | Poll for authorization completion |
| `/api/v1/hytale/oauth/refresh` | POST | Refresh expired access tokens |
| `/api/v1/hytale/oauth/profiles` | POST | List user's game profiles |
| `/api/v1/hytale/oauth/select-profile` | POST | Bind profile to session |
| `/api/v1/hytale/oauth/game-session/new` | POST | Create game session |
| `/api/v1/hytale/oauth/game-session/refresh` | POST | Extend session lifetime |
| `/api/v1/hytale/oauth/game-session/delete` | POST | Terminate session |

### Example: Device Code Flow

```bash
# 1. Request device code
curl -X POST http://localhost:8080/api/v1/hytale/oauth/device-code

# Response:
{
  "device_code": "DE123456789ABCDEF",
  "user_code": "AB12-CD34",
  "verification_uri": "https://accounts.hytale.com/device",
  "expires_in": 1800,
  "interval": 5
}

# 2. User authorizes at verification_uri and enters user_code

# 3. Poll for token (repeat until authorized)
curl -X POST http://localhost:8080/api/v1/hytale/oauth/token \
  -H "Content-Type: application/json" \
  -d '{"device_code": "DE123456789ABCDEF"}'

# Response (after user authorizes):
{
  "access_token": "eyJhbGc...",
  "refresh_token": "refresh_eyJhbGc...",
  "expires_in": 3600,
  "account_id": "550e8400-e29b-41d4-a716-446655440000"
}

# 4. Get profiles and create game session
curl -X POST http://localhost:8080/api/v1/hytale/oauth/profiles \
  -H "Authorization: Bearer eyJhbGc..."

# 5. Create session for selected profile
curl -X POST http://localhost:8080/api/v1/hytale/oauth/game-session/new \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  -d '{"profile_uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479"}'
```

### Documentation

- **[GSP API Reference](docs/HYTALE_API.md)** - Complete API docs with error codes
- **[Downloader CLI Integration](docs/HYTALE_DOWNLOADER_INTEGRATION.md)** - Automated provisioning
- **[Customer Auth Flow](docs/HYTALE_AUTH_FLOW.md)** - User-facing authentication guide

## API Documentation

### Authentication

All API endpoints (except public stats) require authentication:

**API Key Header:**
```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/sync/logs
```

**API Key Query Parameter:**
```bash
curl "http://localhost:8080/api/v1/sync/logs?api_key=your-api-key"
```

**Bearer Token (Admin Routes):**
```bash
curl -H "Authorization: Bearer your-jwt-token" http://localhost:8080/api/admin/settings
```

### Public Endpoints

#### Get System Statistics
```http
GET /api/stats
GET /api/panel/counts
```

**Response:**
```json
{
  "success": true,
  "data": {
    "totalServers": 150,
    "totalUsers": 42,
    "activeUsers": 38,
    "totalAllocations": 500
  }
}
```

### Sync Endpoints (API Key Required)

#### Trigger Full Sync
```http
POST /api/v1/sync/full
Content-Type: application/json
X-API-Key: your-api-key

{
  "skip_users": false,
  "requested_by": "admin@example.com"
}
```

**Response:** `202 Accepted`
```json
{
  "success": true,
  "data": {
    "sync_log_id": "550e8400-e29b-41d4-a716-446655440000",
    "task_id": "asynq:task:abc123def456",
    "status": "PENDING"
  },
  "message": "Full sync has been queued"
}
```

#### Trigger Partial Syncs
```bash
POST /api/v1/sync/locations    # Locations only
POST /api/v1/sync/nodes        # Nodes only
POST /api/v1/sync/servers      # Servers only
POST /api/v1/sync/users        # Users only
```

#### Get Sync Status
```http
GET /api/v1/sync/status/550e8400-e29b-41d4-a716-446655440000
X-API-Key: your-api-key
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "full",
    "status": "RUNNING",
    "itemsTotal": 500,
    "itemsSynced": 250,
    "itemsFailed": 2,
    "error": null,
    "startedAt": "2026-01-09T10:30:00Z",
    "completedAt": null
  }
}
```

#### Get Sync Logs
```http
GET /api/v1/sync/logs?limit=20&offset=0&type=full
X-API-Key: your-api-key
```

#### Cancel Running Sync
```http
POST /api/v1/sync/cancel/550e8400-e29b-41d4-a716-446655440000
X-API-Key: your-api-key
```

### Email Endpoints

#### Queue Email
```http
POST /api/v1/email/queue
Content-Type: application/json
X-API-Key: your-api-key

{
  "to": "user@example.com",
  "subject": "Welcome to NodeByte!",
  "template": "welcome",
  "data": {
    "name": "John Doe",
    "verifyUrl": "https://app.example.com/verify/abc123"
  }
}
```

### Webhook Endpoints

#### Dispatch Webhook
```http
POST /api/v1/webhook/dispatch
Content-Type: application/json
X-API-Key: your-api-key

{
  "event": "sync.completed",
  "data": {
    "syncLogId": "550e8400-e29b-41d4-a716-446655440000",
    "type": "full",
    "status": "COMPLETED",
    "duration": "5m30s"
  }
}
```

### Admin Endpoints (Bearer Token Required)

#### Get System Settings
```http
GET /api/admin/settings
Authorization: Bearer your-jwt-token
```

#### Update System Settings
```http
POST /api/admin/settings
Content-Type: application/json
Authorization: Bearer your-jwt-token

{
  "pterodactylUrl": "https://panel.example.com",
  "autoSyncEnabled": true,
  "autoSyncInterval": 3600
}
```

#### Get Webhooks
```http
GET /api/admin/settings/webhooks
Authorization: Bearer your-jwt-token
```

#### Create Webhook
```http
POST /api/admin/settings/webhooks
Content-Type: application/json
Authorization: Bearer your-jwt-token

{
  "name": "Sync Notifications",
  "webhookUrl": "https://discord.com/api/webhooks/123456/abcdef",
  "type": "SYSTEM",
  "scope": "ADMIN"
}
```

#### Test Webhook
```http
PATCH /api/admin/settings/webhooks
Content-Type: application/json
Authorization: Bearer your-jwt-token

{
  "id": "webhook-id"
}
```

#### Manage Repositories
```bash
# Get repositories
GET /api/admin/settings/repos

# Add repository
POST /api/admin/settings/repos
{
  "repo": "owner/repository"
}

# Update repository
PUT /api/admin/settings/repos
{
  "oldRepo": "old/repo",
  "repo": "new/repo"
}

# Delete repository
DELETE /api/admin/settings/repos
{
  "repo": "owner/repository"
}
```

#### Sync Management
```bash
# Get sync status
GET /api/admin/sync

# Trigger sync
POST /api/admin/sync
{
  "type": "full"  # or: locations, nodes, servers, users
}

# Cancel sync
POST /api/admin/sync/cancel

# Get sync settings
GET /api/admin/sync/settings

# Update sync settings
POST /api/admin/sync/settings
{
  "autoSyncEnabled": true,
  "autoSyncInterval": 3600
}
```

### Statistics Endpoints

```bash
# Overview stats (admin)
GET /api/v1/stats/overview

# Server stats (admin)
GET /api/v1/stats/servers

# User stats (admin)
GET /api/v1/stats/users

# Admin dashboard stats
GET /api/admin/stats
```

## Integration Examples

### Next.js Integration

```typescript
// lib/api.ts
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export async function triggerFullSync(requestedBy: string) {
  const response = await fetch(`${API_BASE}/api/v1/sync/full`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': process.env.BACKEND_API_KEY!,
    },
    body: JSON.stringify({ requested_by: requestedBy, skip_users: false }),
  });

  if (!response.ok) {
    throw new Error(`Sync failed: ${response.statusText}`);
  }

  return response.json();
}

export async function getSyncStatus(syncLogId: string) {
  const response = await fetch(
    `${API_BASE}/api/v1/sync/status/${syncLogId}?api_key=${process.env.BACKEND_API_KEY}`,
  );

  if (!response.ok) {
    throw new Error(`Failed to get sync status`);
  }

  return response.json();
}

// pages/admin/sync.tsx
export default function SyncPage() {
  const [syncLogId, setSyncLogId] = useState<string | null>(null);

  const handleTriggerSync = async () => {
    try {
      const result = await triggerFullSync(session.user.email);
      setSyncLogId(result.data.sync_log_id);
      toast.success('Sync started');
    } catch (error) {
      toast.error('Failed to start sync');
    }
  };

  return (
    <div>
      <button onClick={handleTriggerSync}>Start Full Sync</button>
      {syncLogId && <SyncProgress syncLogId={syncLogId} />}
    </div>
  );
}
```

## Architecture

### Request Flow

```
┌─────────────────────────────────────────────────┐
│         REST API (Fiber Framework)              │
│  Authentication: API Key or Bearer Token        │
└────────────────┬────────────────────────────────┘
                 │
         ┌───────▼────────┐
         │  Route Handler │
         └───────┬────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
┌───▼──┐   ┌────▼─────┐  ┌──▼───┐
│ Sync │   │  Email   │  │Admin │
└───┬──┘   └────┬─────┘  └──┬───┘
    │           │           │
    └────────┬──┴───────┬───┘
             │          │
      ┌──────▼──────────▼──────┐
      │   Queue Manager (Asynq)│
      │  - Critical Queue      │
      │  - Default Queue       │
      │  - Low Queue           │
      └──────┬──────────┬──────┘
             │          │
      ┌──────▼──────────▼───────┐
      │  Worker Processors      │
      │  - SyncHandler          │
      │  - EmailHandler         │
      │  - WebhookHandler       │
      └──────┬──────────┬───────┘
             │          │
      ┌──────▼─────────▼────────┐
      │  External Services      │
      │  - Pterodactyl Panel    │
      │  - Resend Email API     │
      │  - Discord Webhooks     │
      └─────────────────────────┘
```

### Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go                  # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go                # Configuration loading
│   ├── crypto/
│   │   └── encryption.go            # Sensitive data encryption
│   ├── database/
│   │   ├── connection.go            # Database pool management
│   │   ├── webhooks.go              # Webhook repository
│   │   └── sync_repository.go       # Sync log repository
│   ├── handlers/
│   │   ├── api.go                   # API endpoint handlers
│   │   ├── middleware.go            # Authentication middleware
│   │   ├── admin_settings.go        # Admin settings handlers
│   │   ├── admin_webhooks.go        # Webhook management
│   │   ├── errors.go                # Error handling
│   │   └── routes.go                # Route definitions
│   ├── panels/
│   │   └── pterodactyl.go           # Pterodactyl panel API client
│   ├── queue/
│   │   └── manager.go               # Task queue management
│   ├── scalar/
│   │   └── client.go                # Scalar API client
│   └── workers/
│       ├── server.go                # Asynq worker server
│       ├── scheduler.go             # Cron job scheduler
│       ├── sync_handler.go          # Sync task processor
│       ├── email_handler.go         # Email task processor
│       └── webhook_handler.go       # Webhook task processor
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## Monitoring

### Health Checks

```bash
# Check server health
curl http://localhost:8080/health

# Response
{
  "status": "ok",
  "timestamp": "2026-01-09T10:30:00Z"
}
```

### Asynq Web UI

Access the Asynq dashboard for job monitoring:

```bash
# Start with monitoring profile
docker-compose --profile monitoring up -d

# Visit http://localhost:8081
```

### Logging

Logs are structured JSON for easy parsing:

```bash
# View logs
docker-compose logs -f backend

# Example log output
{
  "level": "info",
  "time": "2026-01-09T10:30:00Z",
  "message": "Full sync triggered",
  "sync_log_id": "550e8400-e29b-41d4-a716-446655440000",
  "task_id": "asynq:task:abc123"
}
```

## Performance

- **Sync Operations:** Process 100+ items per batch
- **Job Concurrency:** 10 concurrent workers per queue
- **Connection Pool:** 25 max connections, 5 minimum
- **Request Timeout:** 30 seconds
- **Graceful Shutdown:** 10 second timeout for cleanup

## Development

### Code Standards

- Follow Go conventions and best practices
- Use interfaces for dependency injection
- Comprehensive error handling with zerolog
- Structured logging with contextual information
- Tests for business logic and API handlers
- Code must pass: `gofmt`, `go vet`, `golangci-lint`

# Database Tools - Quick Reference

## One-Time Setup

```bash
cd backend
make build-tools
```

## Common Commands

### Fresh Database
```bash
make db-init
```

### Add New Schemas (Interactive)
```bash
make db-migrate
# Then select schema numbers from menu (e.g., 14,15)
```

### Single Schema
```bash
make db-migrate-schema SCHEMA=schema_15_careers.sql
```

### Start Fresh
```bash
make db-reset
# Confirm by typing database name
```

### See Available Schemas
```bash
make db-list
```

## With Environment Variable

```bash
export DATABASE_URL="postgresql://user:password@localhost:5432/nodebyte"

# Then use commands normally
make db-init
make db-migrate
make db-reset
```

## Direct Binary Usage

```bash
# All commands also work with the binary directly
./bin/db init -database "postgresql://user:password@localhost:5432/nodebyte"
./bin/db migrate -database "postgresql://user:password@localhost:5432/nodebyte"
./bin/db migrate -database "..." -schema schema_15_careers.sql
./bin/db reset -database "postgresql://user:password@localhost:5432/nodebyte"
./bin/db list
./bin/db help
```

## Development Workflow

```bash
# Start fresh
make db-reset
# Confirm: nodebyte
# ✅ Database is now reset and initialized

# Make changes, run tests
# ...

# Add new schema during development
make db-migrate-schema SCHEMA=schema_16_new_feature.sql

# Or choose from menu
make db-migrate
```

## Makefile Targets

```
db-init               # Initialize fresh database
db-migrate            # Interactive schema selection
db-migrate-schema     # Migrate specific schema (SCHEMA=name)
db-reset              # Drop and recreate database
db-list               # List available schemas
build-tools           # Build database tool
```

## Common Issues

**Tool not found?**
```bash
make build-tools
```

**Wrong database connected?**
```bash
export DATABASE_URL="postgresql://user:password@correct-host/correct-db"
make db-migrate
```

**Need to start over?**
```bash
make db-reset
# Type database name to confirm
# Database is now fresh with all 15 schemas
```

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` | PostgreSQL connection string | (none) |
| `SCHEMA` | Used with `db-migrate-schema` | (none) |

## More Information

- **Full Guide**: See `DATABASE_TOOLS.md`
- **Implementation**: See `DATABASE_IMPLEMENTATION.md`
- **Schema Details**: See `schemas/README.md`

---

**Quick Test:**
```bash
make build-tools && make db-list
```

**Help:**
```bash
make help
./bin/db help
```

### Pre-Commit Checks

Before pushing, ensure code passes all checks:

```bash
# Format code
gofmt -w .
goimports -w .

# Run linters
golangci-lint run ./...

# Run tests
go test -v -race ./...

# Build binary
go build -o bin/nodebyte-backend ./cmd/api
```

### Running Tests

```bash
# All tests with race detection
go test -v -race ./...

# With coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # View in browser
```

### Building

```bash
# Build binary (Linux)
go build -o nodebyte-backend ./cmd/api

# Cross-compile (macOS)
GOOS=darwin GOARCH=amd64 go build -o nodebyte-backend ./cmd/api

# Cross-compile (Windows)
GOOS=windows GOARCH=amd64 go build -o nodebyte-backend.exe ./cmd/api
```

## CI/CD Pipelines

Automated workflows run on every commit and PR.

### Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| **Test & Build** | Push/PR | Unit tests, build verification |
| **Lint & Quality** | Push/PR | Code quality (50+ linters) |
| **Format Check** | Push/PR | Code formatting (gofmt, goimports) |
| **Coverage** | Push/PR | Test coverage (70%+ required) |
| **Dependencies** | Weekly | Security scanning, dependency checks |
| **Docker Build** | Tags/Push | Build & push Docker images |

### Local CI Simulation

Run the same checks locally before pushing:

```bash
# Run all checks
./scripts/ci.sh  # If available, or run manually:

gofmt -w .
goimports -w .
golangci-lint run ./...
go test -v -race -coverprofile=coverage.out ./...
go build -o bin/nodebyte-backend ./cmd/api
```

### GitHub Actions Status

Check workflow status:
- **GitHub Web:** Actions tab
- **CLI:** `gh run list` / `gh run watch <id>`
- **Email:** Failed workflow notifications


## Troubleshooting

### Connection Issues

```bash
# Test database connection
psql $DATABASE_URL -c "SELECT 1"

# Test Redis connection
redis-cli -u $REDIS_URL ping

# Check if server is running
curl -v http://localhost:8080/health
```

### Worker Not Processing Jobs

```bash
# Check Asynq Web UI at http://localhost:8081 (if monitoring profile active)
# Or check logs for worker errors
docker-compose logs backend | grep -i error

# Verify queue connectivity
redis-cli -u $REDIS_URL KEYS "*"
```

### Sync Not Completing

1. Check Pterodactyl API credentials in `.env`
2. Verify network connectivity to Pterodactyl panel
3. Review sync logs: `GET /api/v1/sync/logs` (requires API key)
4. Check worker server status in Asynq UI
5. Look for database connection pool exhaustion in logs

### Hytale OAuth Issues

1. Verify `HYTALE_ENVIRONMENT` is set correctly (production or staging)
2. Check JWKS cache refresh
3. Review audit logs for auth failures
4. Ensure database has `hytale_audit_logs` table

### Code Quality Issues

**"gofmt errors"**
```bash
gofmt -w .
goimports -w .
```

**"golangci-lint errors"**
```bash
golangci-lint run ./...  # View all issues
# Fix issues in code, then retry
```

**"Test coverage below 70%"**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # View coverage gaps
# Write tests for uncovered code
```

## Architecture Overview

```
┌────────────────────────────────────┐
│   REST API (Fiber v2 Framework)    │
│  Auth: API Key, Bearer, or Public  │
└────────────────┬───────────────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
┌───▼──┐    ┌───▼────┐  ┌────▼────┐
│ Sync │    │ Hytale │  │  Admin  │
│      │    │ OAuth  │  │Settings │
└───┬──┘    └───┬────┘  └────┬────┘
    │           │            │
    └───────────┼────────────┘
                │
        ┌───────▼──────────┐
        │  Queue Manager   │
        │  (Asynq/Redis)   │
        └────────┬─────────┘
                 │
        ┌────────▼────────┐
        │    Workers      │
        │- Sync Handler   │
        │- Email Handler  │
        │- Webhook Handler│
        │- OAuth Refresher│
        │- Scheduler      │
        └────────┬────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
┌───▼──┐   ┌────▼──┐   ┌────▼────┐
│ Panel│   │ Hytale│   │ Resend  │
│      │   │ OAuth │   │ Email   │
└──────┘   └───────┘   └─────────┘
```

## Performance

- **Sync Operations:** Process 100+ items per batch
- **Job Concurrency:** 10 concurrent workers per queue
- **Connection Pool:** 25 max connections, 5 minimum
- **Rate Limiting:** Token bucket (5-20 req/min per endpoint)
- **Token Refresh:** Auto-refresh every 5-10 minutes
- **JWKS Cache:** Hourly refresh, ~99% hit rate
- **Request Timeout:** 30 seconds
- **Graceful Shutdown:** 10 second timeout for cleanup

## Related Documentation

- [Hytale GSP API Reference](docs/HYTALE_API.md) - Complete API docs
- [Downloader CLI Integration](docs/HYTALE_DOWNLOADER_INTEGRATION.md) - Provisioning guide
- [Customer Auth Flow](docs/HYTALE_AUTH_FLOW.md) - User authentication guide
- [CHANGELOG.md](CHANGELOG.md) - Version history and features

## Support & Community

- **Issues & Bugs:** Report on GitHub Issues
- **Documentation:** See inline code comments and docs/
- **Discord:** Join our community server
- **Email:** support@nodebyte.com

## License

AGPL 3.0 - See [LICENSE](LICENSE) file for details