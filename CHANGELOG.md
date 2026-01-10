# Changelog

All notable changes to NodeByte Backend will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Partial sync capabilities (sync specific resource types)
- Webhook event filtering and delivery guarantees
- Prometheus metrics export
- gRPC API for internal communication
- Support for additional panel integrations (Game Panel Pro, Wings)

## [0.1.0] - 2026-01-09

### Added

#### Encryption System
- AES-256-GCM encryption for sensitive configuration values
- `crypto.Encryptor` implementation with base64-encoded key support
- Support for encrypted storage of API keys in database (`config` table)
- Graceful fallback when encryption key not configured (unencrypted mode)
- Environment variable support for ENCRYPTION_KEY configuration

#### Sync Logging & Tracking Improvements
- **Detailed progress tracking** via `updateDetailedProgress()` 
  - Per-step item totals and completion counts
  - Real-time percentage completion calculation
  - Detailed status messages with contextual information
- **Sync log database updates**
  - `itemsTotal` field now properly populated for each sync step
  - `itemsSynced` tracks successful operations
  - `itemsFailed` tracks failed items (when applicable)
  - `completedAt` timestamp recorded on completion
- **Metadata storage** with JSON serialization
  - `step`: current sync operation (locations, nodes, allocations, etc.)
  - `itemsTotal` and `itemsProcessed`: progress tracking
  - `percentage`: calculated completion percentage
  - `lastMessage`: detailed human-readable status message
  - `lastUpdated`: timestamp of last status update

#### Webhook Notifications
- **Sync completion webhooks** - Automatic Discord webhook dispatch on sync finish
- **Status notification system**:
  - ✅ Success notifications with duration and item counts
  - ❌ Failure notifications with error details
  - ⚠️ Cancellation notifications
- **Rich Discord embeds** with:
  - Operation status and emoji indicators
  - Execution duration in seconds
  - Error messages (when applicable)
  - Timestamp and footer attribution
- **Background webhook dispatch** using goroutines with background context

#### Admin Settings & Configuration
- **Settings change tracking** with before/after value diffs
- **Admin audit trail** - Track which admin made configuration changes
- **Webhook management endpoints**:
  - GET/POST/PUT/DELETE for Discord webhook CRUD operations
  - Webhook testing functionality
  - Type and scope filtering (SYSTEM, GAME_SERVER, VPS, etc.)
- **Webhook test endpoint** for validating Discord connectivity

#### Pterodactyl API Enhancements
- **Dual API key support** - Application and Client API key configuration
- **Client API methods** via `doClientRequest()` for user-accessible endpoints
- **Server resource tracking**:
  - `GetServerResources()` - Live CPU, memory, disk, and network usage
  - `GetServerDetailWithIncludes()` - Detailed server info with specified relationships
- **Proper authorization headers** - "Bearer {apiKey}" format for all requests

#### Core Infrastructure
- Initial Go backend service scaffolding with modular architecture
- HTTP server using Fiber v2.52.5 framework with graceful shutdown
- Redis-backed job queue system using Asynq v0.24.1
- PostgreSQL connection pooling with pgx v5.7.2
- Structured JSON logging with zerolog v1.33.0
- Cron scheduler using robfig/cron v3.0.1 for recurring background jobs
- Docker containerization with docker-compose for development environment
- Cross-platform Makefile with Windows compatibility
- Comprehensive environment configuration management

#### Configuration System
- Environment-based configuration loader for connection strings and API keys
- Database-aware config merging via `Config.MergeFromDB()` method
- System settings loaded from database `config` table at startup
- Support for Pterodactyl panel URL, API key, and credentials management
- Resend email service API key configuration
- Discord webhook configuration for notifications
- Feature flags and sync settings stored in database
- Redis URL parser supporting both `host:port` and `redis://user:pass@host:port/db` formats

#### Database Layer
- PostgreSQL connection pool with configurable pool size
- Database models for all Pterodactyl entities:
  - Locations, Nodes, Allocations
  - Nests, Eggs, Egg Variables
  - Servers, Server Databases
  - Users, User Permissions
  - Sync logs for tracking migration progress
- SyncRepository for managing sync operations and logs
- UPSERT patterns with ON CONFLICT clauses for atomic operations
- Foreign key relationships and data integrity constraints
- Auto-generated UUID primary keys for internal records

#### Pterodactyl API Client
- Comprehensive HTTP wrapper for Pterodactyl panel API v1
- Response type definitions matching Pterodactyl API structure:
  - Locations with region data
  - Nodes with resource and daemon configuration
  - Allocations with IP/port mapping
  - Nests with author metadata
  - Eggs with variable definitions
  - Servers with resource limits and status
  - Server databases with host information
  - Users with admin role mapping
- Automatic pagination support via `getAllWithPagination()` helper
- Relationship includes support (e.g., server allocations, egg variables)
- Query parameter merging for complex filter requests
- Custom unmarshal callbacks for type-safe response handling

#### Sync Handlers (Complete Implementation)
Comprehensive background job handlers for synchronizing Pterodactyl infrastructure:

- **HandleFullSync()** — Orchestrated full data sync pipeline
  - Sequential sync of locations → nodes → allocations → nests/eggs → servers → users
  - Real-time progress tracking (0% → 100%)
  - Cancellation support between sync steps
  - Detailed error handling with failure logs
  - Optional user sync skipping

- **HandleSyncLocations()** — Location synchronization
  - Fetches all locations from Pterodactyl
  - Maps region/location data to local database
  - Upserts location metadata (short_code, description)

- **HandleSyncNodes()** — Node synchronization
  - Fetches all compute nodes with resource configuration
  - Stores 18+ node attributes: UUID, FQDN, scheme, memory/disk limits
  - Maintains daemon configuration (listen port, SFTP port, base URL)
  - Links nodes to locations

- **HandleSyncAllocations()** — Server port allocation synchronization
  - Iterates all nodes and fetches their port allocations
  - Handles large allocation sets efficiently with batch inserts (500 records per query)
  - Tracks IP/port assignment status
  - Supports allocation aliases and notes

- **HandleSyncNests()** — Game/Software nest synchronization
  - Syncs available game/software templates
  - Fetches associated eggs (server images/configurations)
  - **Syncs egg variables** with user-editable config options
  - Three-level hierarchy: Nest → Egg → Variables
  - Preserves variable validation rules

- **HandleSyncServers()** — Server synchronization (most complex)
  - Fetches all hosted servers with full metadata
  - Status mapping (online/offline/suspended)
  - Links servers to owner users via pterodactyl_id
  - Maps resource limits: memory, disk, CPU allocation
  - Tracks server name, description, UUID, external ID
  - Handles server state changes via ON CONFLICT updates

- **HandleSyncDatabases()** — Server database synchronization
  - Syncs managed databases for each server
  - Includes database host information
  - Tracks connection limits and credentials
  - Links databases to parent servers

- **HandleSyncUsers()** — Pterodactyl user synchronization
  - Fetches all panel users with pagination
  - Creates or updates user records based on email
  - Preserves existing local user data (password hash, tokens)
  - Updates Pterodactyl-sourced fields: pterodactyl_id, admin flag
  - Supports admin role mapping

- **HandleCleanupLogs()** — Sync log maintenance
  - Removes sync logs older than configured threshold (default: 30 days)
  - Prevents database bloat from historical records

#### Job Queue
- Task enqueueing for sync operations with payload serialization
- Support for full sync, incremental syncs, and cleanup tasks
- Payload types for flexible sync configuration
- Error handling with retry logic via Asynq
- Job visibility and monitoring

#### API Endpoints
- REST API routes with Fiber routing system
- Sync status checking and progress reporting
- Admin endpoints for manual sync triggering
- Health check endpoints
- Error response formatting with descriptive messages

#### Error Handling & Logging
- Structured logging with context (sync_log_id, job_id, error details)
- Per-record error logging with continuation on failures
- Graceful degradation (one failed record doesn't stop entire sync)
- Database connection error recovery
- Redis connection validation and reconnection logic
- Detailed error messages for debugging

### Changed

- **API key handling** - Keys now decrypted from database before Pterodactyl client initialization
- **Webhook dispatch timing** - Now uses `context.Background()` to prevent "context canceled" errors
- **Sync log creation** - Now includes type, status, and metadata initialization
- **UpdateSyncLog function signature** - Added `itemsTotal` parameter for complete progress tracking
- **Configuration loading order** - Load from env first, then override with decrypted database values
- Extracted sync service from Next.js API routes to standalone Go service
- Moved from synchronous API request/response to async job queue pattern
- Transitioned from TypeScript/Node.js to Go for better performance and resource efficiency
- Configuration management from environment-only to database-aware system

### Fixed

- **Webhook context cancellation** - Dispatched in goroutines with fresh background context
- **API key encryption/decryption pipeline** - Keys properly decrypted during config merge
- **Sensitive field masking** - API keys and tokens masked in API responses
- **Sync progress calculation** - Accurate percentage based on itemsTotal and itemsProcessed
- **Migration support** - Graceful handling of existing unencrypted values
- **Decryption failure handling** - Warning messages when decryption fails, fallback to raw value
- Go cron specification syntax (strconv.Itoa for decimal string conversion)
- Asynq error handler callback signature (context.Context instead of asynq.Context)
- Windows Makefile compatibility (removed ANSI escape sequences, Unix shell syntax)
- Redis connection URL parsing for both simple and complex URIs
- Pterodactyl API response type marshaling with proper relationship handling
- Environment file loading from relative paths

### Technical Details

- **Language**: Go 1.23
- **HTTP Framework**: Fiber v2.52.5
- **Job Queue**: Asynq v0.24.1 (Redis-backed)
- **Database Driver**: pgx v5.7.2 (PostgreSQL)
- **Scheduler**: robfig/cron v3.0.1
- **Logging**: zerolog v1.33.0
- **Container Runtime**: Docker with docker-compose
- **Build System**: Cross-platform Makefile (Windows, Linux, macOS)

### Architecture Decisions

- **Async-First Design**: All long-running operations (syncs, email, webhooks) queued as background jobs
- **Pagination Abstraction**: Generic pagination helper in Pterodactyl client eliminates manual page loops
- **Database-Driven Config**: System settings managed via admin panel (database) rather than environment variables
- **Upsert Pattern**: ON CONFLICT clauses for reliable idempotent syncs
- **Error Resilience**: Individual record failures don't cascade to entire sync operation
- **Structured Logging**: JSON logs for easy parsing and monitoring
- **Modular Handlers**: Separate handlers for each sync type allow independent testing and parallel execution
- **Encryption First**: Sensitive values encrypted at rest with database-managed keys

### Security

- **Encrypted API keys at rest** - All sensitive credentials encrypted in database using AES-256-GCM
- **Key rotation support** - Can change ENCRYPTION_KEY and re-encrypt values
- **Sensitive field detection** - Automatic identification of fields requiring encryption:
  - pterodactyl_api_key
  - pterodactyl_client_api_key
  - virtfusion_api_key
  - resend_api_key
  - cf_access_client_secret
  - scalar_api_key
- **Audit trail** - Admin changes logged with username and timestamp
- API key validation via BACKEND_API_KEY environment variable
- CORS origin validation with configurable whitelist
- Prepared statements for all SQL queries (preventing injection)
- Pterodactyl panel credentials stored in database (not environment)
- Graceful error messages without sensitive data exposure

### Performance

- **Allocations batch insert** - Changed from individual INSERT to 500-record batches (~100x faster)
- **Improved webhook dispatch** - Non-blocking background goroutines with retry capability
- Connection pooling for database (pgxpool) and Redis
- Automatic pagination for large datasets (locations, servers, users, allocations)
- Bulk upserts with single queries per record type
- Background job processing prevents blocking API responses
- Cron-based scheduled syncs with configurable intervals
- Efficient type assertions for relationship unmarshaling

---

[Unreleased]: https://github.com/NodeByteHosting/nodebyte-host/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/NodeByteHosting/nodebyte-host/releases/tag/v0.1.0
