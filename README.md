# NodeByte Backend API

REST API and background job processing service for NodeByte infrastructure management.

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-AGPL%203.0-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-brightgreen.svg)](Dockerfile)

## Overview

The NodeByte Backend API provides a comprehensive REST API for managing game server infrastructure, with features including:

- **Panel Synchronization** - Full Pterodactyl panel sync (locations, nodes, allocations, nests, eggs, servers, users, databases)
- **Job Queue System** - Redis-backed async job processing with priority queues
- **Admin Dashboard** - Complete REST API for system settings, webhooks, and sync management
- **Email Queue** - Asynchronous email sending via Resend API
- **Discord Webhooks** - Real-time notifications for sync events and system changes
- **Cron Scheduler** - Automated scheduled sync jobs with configurable intervals
- **Database Configuration** - Dynamic system settings management with encryption support
- **Health Monitoring** - Built-in health checks and comprehensive structured logging

## Quick Start

### Local Development

```bash
# Clone repository
cd backend

# Install dependencies
go mod download
go mod tidy

# Create environment file
cp .env.example .env

# Edit configuration
nano .env

# Start dependencies (Redis + PostgreSQL)
docker-compose up -d postgres redis

# Run server
go run ./cmd/api/main.go

# Server runs at http://localhost:8080
```

### Docker Deployment

```bash
# Start all services
docker-compose up -d

# With monitoring (Asynq UI)
docker-compose --profile monitoring up -d

# View logs
docker-compose logs -f backend

# Shutdown
docker-compose down
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

# Pterodactyl Panel
PTERODACTYL_URL=https://panel.example.com          # Required
PTERODACTYL_API_KEY=your-admin-api-key             # Required
PTERODACTYL_CLIENT_API_KEY=client-api-key          # Optional

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

The backend uses the same PostgreSQL database as the main application. Required tables are created automatically on first run. See the main app's migrations for schema details.

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

### Running Tests

```bash
go test ./...
go test -race ./...  # With race detection
```

### Building

```bash
# Build binary
go build -o nodebyte-backend ./cmd/api

# Cross-compile (macOS)
GOOS=darwin GOARCH=amd64 go build -o nodebyte-backend ./cmd/api

# Cross-compile (Linux)
GOOS=linux GOARCH=amd64 go build -o nodebyte-backend ./cmd/api
```

## Troubleshooting

### Connection Issues

```bash
# Test database connection
psql $DATABASE_URL -c "SELECT 1"

# Test Redis connection
redis-cli -u $REDIS_URL ping
```

### Worker Not Processing Jobs

```bash
# Check Asynq Web UI at http://localhost:8081
# Or check logs for worker errors
docker-compose logs backend | grep -i error
```

### Sync Not Completing

1. Check Pterodactyl API credentials
2. Verify network connectivity to panel
3. Review sync logs: `GET /api/v1/sync/logs`
4. Check worker server status in Asynq UI

## Support

- **Issues:** Report bugs on GitHub
- **Documentation:** See inline code comments and API docs
- **Discord:** Join our community server

## License

AGPL 3.0 - See LICENSE file for details