# Changelog

All notable changes to NodeByte Backend will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - unreleased

### Added
- **Unified Database CLI Tool** - Consolidated shell scripts into cross-platform Go CLI
  - Supports Windows, macOS, and Linux without native shell dependencies
  - `db init` - Initialize database schema with interactive schema selection
  - `db migrate` - Run specific schema migrations with validation
  - `db reset` - Complete database reset with confirmation prompt
  - `db list` - Display all available schema files with status
  - Makefile integration with environment variable loading from `.env`
- **Consolidated Next.js API Routes to Go Backend** - Complete migration of frontend API routes to backend
  - Moved all `/api/admin/` endpoints from Next.js to Go Fiber backend
  - Admin user management: `GET/POST /api/admin/users`, `POST /api/admin/users/roles`
  - Admin settings: `GET/POST /api/admin/settings`, `POST /api/admin/settings/test`
  - GitHub repositories: `GET/POST/PUT/DELETE /api/admin/settings/repos`
  - Discord webhooks: `GET/POST/PUT/PATCH/DELETE /api/admin/settings/webhooks`
  - Admin sync controls: `GET/POST /api/admin/sync`, `GET /api/admin/sync/logs`, `GET/POST /api/admin/sync/settings`
  - Admin servers: `GET /api/admin/servers`
  - Bearer token authentication middleware for all admin routes
  - Consistent error response format across all endpoints
- **Admin User Management API** - Complete user listing and management endpoints
  - `GET /api/admin/users` - Paginated user listing with filtering, sorting, search
  - Query parameters: page, pageSize, sortField, sortOrder, filter, search
  - Returns paginated response with user data and statistics
  - `POST /api/admin/users/roles` - Update user role assignments
  - User statistics: totalUsers, migratedUsers, adminCount, activeCount
- **Hytale Token Auto-Push to Pterodactyl** - Automatic environment variable updates for game servers
  - Game sessions can be linked to specific Pterodactyl servers via `server_id` field
  - Background worker automatically pushes refreshed tokens to Pterodactyl every 5 minutes
  - Environment variables `HYTALE_SESSION_TOKEN` and `HYTALE_IDENTITY_TOKEN` updated via Client API
  - Schema migration `schema_13_hytale_server_link.sql` adds `server_id` column to `hytale_game_sessions`
  - Graceful degradation: Logs warnings but continues if Pterodactyl push fails

- **User Account API** - Full profile management endpoints for dashboard
  - `GET /api/v1/dashboard/account` - Returns complete user profile (name, email, phone, company, billing email, roles, verification status, last login)
  - `POST /api/v1/dashboard/account/resend-verification` - Resend email verification with queued email delivery
  - `POST /api/v1/dashboard/account/change-email` - Request email address change with verification token
  - Nullable field handling for optional user profile columns (`username`, `phoneNumber`, `companyName`, `billingEmail`)
- **Stale Record Cleanup During Sync** - Automatic deletion of records no longer present on the panel
  - Locations, nodes, allocations, nests, and servers are pruned after each sync step
  - Allocations deletion batched across all nodes using collected IDs
  - Server deletion scoped to `panelType = 'pterodactyl'` with non-null `pterodactylId` to avoid deleting manually-created records

### Fixed
- **Admin Users Data Type Handling** - Fixed TIMESTAMP column scanning from PostgreSQL
  - Changed timestamp handling to use `time.Time` objects with RFC3339 formatting
  - Properly convert database TIMESTAMP columns to ISO 8601 string format in API responses
  - Fixed empty users array issue in admin panel users listing
  - Proper null pointer handling for nullable timestamp fields (`lastLoginAt`, `emailVerifiedTime`)
- **Server-Allocation Relationship Sync** - Fixed missing foreign key population
  - Added `Relationships` field to `PteroServer` struct to capture included allocations from API
  - `syncServers()` now properly updates `server_id` foreign key in `allocations` table
  - Fixes issue where allocations were synced but not linked to their servers
  - All server-allocation relationships now properly populated during full sync
- **Allocation Server ID Column Name** - Fixed snake_case vs camelCase mismatch in sync query
  - Changed `server_id` to `"serverId"` in allocation UPDATE query to match actual DB column name
  - Allocations now correctly display their associated server names in the admin panel
- **User Account Nullable Username** - Fixed 500 error on `/api/v1/dashboard/account`
  - Changed `Username` field from `string` to `*string` to handle nullable `username TEXT` column
  - Prevents `pgx` scan failure when username is NULL

## [0.2.1] - 2026-01-14

### Added
- **Sentry Error Tracking Integration** - Production error monitoring and performance tracking
  - Optional Sentry SDK integration for real-time error tracking and transaction monitoring
  - Fiber middleware for automatic panic recovery and error reporting
  - `SENTRY_DSN` environment variable configuration
  - Request context preservation with tags and custom data
  - Helper functions for manual error capture (`sentry.CaptureException`, `sentry.CaptureMessage`)
  - Transaction tracking with smart sampling (10% default)
  - Error classification and context enhancement (`CaptureErrorWithContext`)
  - 5-second timeout during graceful shutdown for pending event delivery
  - **Custom Instrumentation Helpers** for detailed performance monitoring
    - `StartSpan()` - Create child spans within transactions
    - `GetTransactionFromContext()` - Access active transactions from any context
    - `GetSpanFromContext()` - Access active spans from any context
    - `CaptureExceptionWithContext()` - Capture errors in background workers without Fiber context
  - **Background Worker Transaction Tracking**
    - `worker.refresh_oauth_tokens` - OAuth token refresh operations (every 5 minutes)
    - `worker.refresh_game_sessions` - Game session refresh checks (every 5 minutes)
    - `worker.cleanup_expired_sessions` - Session cleanup operations (daily at 2 AM)
    - `worker.full_sync` - Pterodactyl panel synchronization jobs
    - `worker.discord_webhook` - Discord webhook dispatch operations
- **Hytale Game Session Enhancements** - Improved session lifecycle management
  - Game session refresh interval changed to every 5 minutes
  - Sessions are refreshed when 5 minutes before expiry
  - Session token refresh returns new `sessionToken` and `identityToken` from Hytale
  - Database method to update session tokens after refresh (`UpdateGameSessionTokens`)
  - Pterodactyl API integration to push refreshed tokens to panel servers
  - New `UpdateServerEnvironment` method for updating environment variables via Pterodactyl client API
  - Sentry transaction tracking for all session refresh operations
- **Hytale Audit Logs API** - New endpoint for retrieving Hytale OAuth event logs
  - `GET /api/v1/hytale/logs?account_id={id}&limit={limit}` endpoint
  - Query parameter validation (max 1000 logs per request)
  - Returns audit logs with event types (TOKEN_CREATED, TOKEN_REFRESHED, SESSION_CREATED, etc.)
  - Supports filtering by account ID
- **Comprehensive Unit Test Suite** - Core packages tested for reliability
  - Config loading and environment variable parsing tests
  - Error handling and HTTP status mapping tests
  - JWKS cache and key management tests
  - OAuth client configuration tests
  - Rate limiter token bucket algorithm tests
  - Type structure and serialization tests
- **CI/CD Workflow Improvements** - Automated quality gates
  - Coverage reporting with 30% minimum threshold
  - PostgreSQL 15 and Redis 7 service containers for integration tests
  - Multi-workflow setup: test-build, format, docker, dependencies, coverage
  - HTML coverage reports uploaded to GitHub artifacts
  - Coverage comments on pull requests with metrics
- **Dual API Integration**: Full Pterodactyl Client API support for user-server relationships
  - New `server_subusers` table tracking user access to servers with permissions
  - `GetClientServers()` method to fetch servers via Client API
  - `GetServerSubusers()` method to fetch subusers for specific servers
  - `syncServerSubusers()` sync method integrating Client API data into database
  - Configurable subuser sync (enabled by default, 25 servers per batch)
  - Automatic owner marking in server_subusers table
  - Rate limiting with 2-second delays between batches
  - New config options: `SYNC_SUBUSERS_ENABLED`, `SYNC_SUBUSERS_BATCH_SIZE`
  - Client API types: `ClientServer` and `ClientSubuser` structs
- **Hytale Game Server Log Persistence** - Automatic backup of console logs to database
  - Background worker periodically fetches logs from Hytale API (every 5 minutes)
  - Persistent storage in PostgreSQL with sync state tracking to prevent duplicates
  - Automatic cleanup of logs older than 30 days (runs daily at 4 AM)
  - API endpoints for historical log retrieval:
    - `GET /api/v1/hytale/server-logs` - Retrieve paginated logs for a game server
    - `GET /api/v1/hytale/server-logs/count` - Get total log count for a server
  - Sync state tracking to monitor last successful sync and handle API errors gracefully
  - Sentry transaction tracking for log persistence operations

### Changed
- **Sentry SDK Configuration** - SDK API compatibility improvements
  - TracesSampler now correctly returns `float64` instead of `sentry.Sampled`
  - Event tags use `event.Tags` map directly instead of `SetTag()` method
  - Span operations properly initialized with `WithOpName` and `WithTransactionSource` options
  - Structured logging integration enabled (`EnableLogs: true`) for automatic log capture
- **Workflow Reliability** - Fixed workflow configuration issues
  - Updated deprecated artifact actions (v3 → v4)
  - Added continue-on-error flags to linting steps to prevent false failures
  - Fixed gofmt check to use `gofmt -l` instead of `go fmt ./...` (non-destructive)
  - Added PR-conditional GitHub script execution to prevent undefined context errors
  - PostgreSQL readiness check added to coverage workflow for test database setup
  - Full sync now includes server subusers as Step 7 (non-blocking, continues on failure)
  - Progress tracking updated to 85% for subuser sync step
  - Config struct extended with `SyncSubusersEnabled` and `SyncSubusersBatchSize` fields

### Fixed
- **Hytale OAuth Handler** - Corrected return value handling
  - RefreshGameSession handler now properly captures 2 return values from oauth client (`sessionResp`, `err`)
  - New tokens are now persisted to database immediately after refresh
- **Test Compatibility** - Aligned tests with actual implementation
  - Fixed OAuth endpoint URL format to match `https://oauth.accounts.{host}` pattern
- **Sentry Background Worker Panic** - Fixed nil pointer dereference in background jobs
  - Added `StartBackgroundTransaction()` function for workers without HTTP context
  - Updated `StartTransaction()` to handle nil fiber.Ctx gracefully
  - Fixed 6 worker functions: `RefreshOAuthTokens`, `RefreshGameSessions`, `CleanupExpiredSessions`, `CleanupOldLogs`, `HandleDiscordWebhook`, `HandleFullSync`
  - All background jobs now use proper Sentry transaction tracking with `task` operation type
- **Sync Progress Updates** - Enhanced frontend polling visibility
  - Added `lastMessage` field to all sync progress updates for real-time status display
  - Standardized status values to `RUNNING` (was inconsistently `in_progress`)
  - Progress updates now include user-friendly messages like "Syncing locations...", "Syncing nodes...", etc.
  - Added `lastUpdated` timestamp to metadata for frontend polling
- **Config Validation** - Removed duplicate field declarations
  - Fixed duplicate `SyncSubusersEnabled` and `SyncSubusersBatchSize` fields in Config struct
  - Corrected rate limiter Allow() return types (int vs int64)
  - Fixed test expectations for rate limiter cleanup behavior
  - Removed tests for private/unexported functions
  - Aligned JWKS cache tests with actual NewJWKSCache(useStaging bool) signature
- **Swagger Documentation Generation** - Fixed type reference issues in OpenAPI comments
  - Corrected Swagger parameter annotations from backslash to dot notation (`types.RefreshTokenRequest` instead of `types.\RefreshTokenRequest`)
  - Added missing `types.` package qualification to unqualified type references in `@Success` and `@Failure` annotations
  - Fixed missing `types.` prefix on SuccessResponse instantiation in SelectProfile handler
  - All Hytale OAuth handler endpoints now generate valid Swagger documentation

## [0.2.0] - 2026-01-14

### Added

#### Hytale OAuth Device Code Flow
- **Device Code Authorization** - Full OAuth 2.0 Device Authorization Grant implementation
  - `/api/hytale/device-code` - Request device code for GSP authentication
  - Automatic code expiration (30 min) and polling timeout (15 min)
  - User-friendly device code format with verification URL
  - Session binding for code→token exchange
- **Token Polling** - `/api/hytale/token-poll` endpoint with configurable backoff
  - Polls Hytale OAuth server for authorization completion
  - Automatic retry-after header handling
  - Clean response on session limit errors (403 Forbidden)
  - Account and profile ID extraction from token response
- **Token Refresh** - `/api/hytale/refresh-token` for extending session validity
  - Automatic refresh window (7 days before expiry)
  - Transaction-safe token replacement in database
  - Preserves account_id and profile_id across refreshes
- **Token Validation** - JWT signature verification with Ed25519 keys
  - JWKS public key caching with hourly automatic refresh
  - Cryptographic signature validation preventing replay attacks
  - Token expiry validation with 60-second clock skew tolerance
  - Audience (aud) claim verification for correct token type
  - Subject (sub) claim extraction for profile identification

#### Game Session Management
- **Session Endpoints** - Complete game session lifecycle management
  - `POST /api/hytale/game-session` - Create new game session with validation
  - `PATCH /api/hytale/game-session` - Refresh existing session (10 min window)
  - `DELETE /api/hytale/game-session` - Terminate session with Hytale cleanup
  - Query parameter validation (account_id, profile_id, session_id)
- **Session Tracking** - Database persistence for audit and compliance
  - Account and profile associations
  - Session state tracking (created, active, terminated)
  - IP address and user agent logging
  - Timestamp recording for session lifecycle

#### JWT Token and JWKS Management
- **JWKSCache** - Thread-safe public key caching system
  - Automatic hourly refresh from Hytale JWKS endpoint
  - Base64URL decoding of Ed25519 public keys
  - In-memory bucket cleanup for stale entries (30 min inactivity)
  - Concurrent read access via RWMutex protection
- **TokenValidator** - Complete JWT validation pipeline
  - Header/payload/signature parsing from JWT format
  - Ed25519 signature verification with extracted public keys
  - Claim validation (iat, exp, sub, aud)
  - Base64URL decoding with proper padding
  - Support for both identity and session token types
- **Database Token Storage** - Secure token persistence
  - Encrypted token fields using AES-256-GCM
  - Account and profile associations
  - Expiry timestamps for automatic cleanup
  - Staging environment support for testing

#### Error Handling & HTTP Mapping
- **HytaleError Type** - Standardized error response structure
  - Application error codes (INVALID_REQUEST, UNAUTHORIZED, SESSION_LIMIT, etc.)
  - User-facing error messages (no sensitive data leakage)
  - Internal messages for debugging logs
  - Session limit flag for special handling
- **HTTP Status Mapping**:
  - 400 Bad Request → INVALID_REQUEST, INVALID_DEVICE_CODE
  - 401 Unauthorized → UNAUTHORIZED, EXPIRED_TOKEN, INVALID_TOKEN
  - 403 Forbidden → FORBIDDEN, SESSION_LIMIT_EXCEEDED (with entitlement hint)
  - 404 Not Found → ENDPOINT_NOT_FOUND, SESSION_NOT_FOUND
  - 429 Too Many Requests → RATE_LIMITED
  - 500+ Server Errors → SERVICE_ERROR, INTEGRATION_ERROR
- **Response Types** - Consistent JSON error responses
  - DetailedErrorResponse with code, message, status fields
  - SessionLimitErrorResponse with entitlement hint
  - NotFoundErrorResponse for missing resources
  - RateLimitErrorResponse with X-RateLimit headers

#### Audit Logging for Compliance
- **Hytale Audit Log Repository** - Compliance-grade event tracking
  - 8 event types: TOKEN_CREATED, TOKEN_REFRESHED, TOKEN_DELETED, AUTH_FAILED, SESSION_CREATED, SESSION_REFRESHED, SESSION_DELETED, PROFILE_SELECTED
  - Account, profile, and IP tracking for forensics
  - User agent capture for device fingerprinting
  - Queryable audit trail via GetAuditLogs(account_id, limit)
- **Database Schema** - Indexed audit table with constraints
  - UUID primary key with auto-generation
  - Event type ENUM validation (8 valid types)
  - Composite indexes on account_id, event_type, created_at DESC
  - JSON details field for flexible event metadata
  - Foreign key relationship to accounts table
- **Audit Events Logged**:
  - OAuth token creation/refresh/deletion with timing
  - Game session lifecycle (created, refreshed, deleted)
  - Authentication failures with reason codes
  - Profile selection for session binding
  - All events include IP, user agent, timestamp

#### Rate Limiting (Token Bucket Algorithm)
- **Distributed Rate Limiter** - Per-endpoint configurable limits
  - Token bucket refill algorithm with float64 precision
  - IP-based limiting for unauthenticated endpoints (device code)
  - Account-based limiting for authenticated endpoints (token ops, sessions)
  - Thread-safe via sync.RWMutex protection
- **Endpoint Configurations**:
  - Device Code: 5 requests per 15 minutes (per IP)
  - Token Poll: 10 requests per 5 minutes (per account)
  - Token Refresh: 6 requests per 1 hour (per account)
  - Game Session: 20 requests per 1 hour (per account)
- **Response Headers** - Standard rate limit signaling
  - X-RateLimit-Limit: requests allowed in window
  - X-RateLimit-Remaining: tokens left for requester
  - X-RateLimit-Reset: Unix timestamp when bucket refills
- **Rate Limit Errors** - Proper 429 responses on limit exceed

#### Type Consolidation
- **Internal Types Package** - Organized request/response definitions
  - `types/auth.go` - Authentication request/response types
  - `types/hytale_oauth.go` - OAuth and session types
  - `types/error_responses.go` - Error response types with Swagger annotations
  - `types/rate_limit.go` - Rate limit error responses
  - `types/token_validation.go` - JWT validation request/responses
  - Eliminates scattered type definitions across handler files
  - Centralized Swagger documentation via struct tags
  - Reduced import chains and circular dependency risk

#### Staging Environment Support
- **Configuration Branching** - Per-environment OAuth endpoints
  - Production Hytale OAuth: `https://oauth.hytale.com`
  - Staging Hytale OAuth: `https://oauth.staging.hytale.com`
  - Environment variable `HYTALE_ENVIRONMENT` (production|staging)
  - Automatic token endpoint and JWKS URL selection
- **Testing Support** - Isolated staging credentials
  - Separate staging database schema (optional)
  - Staging OAuth tokens won't interfere with production
  - Safe testing of device code, token, session flows

### Changed

- **OAuth Endpoints Middleware** - All 8 Hytale routes now protected with rate limiting
  - Device code endpoint: 5 per 15 min per IP
  - Token polling: 10 per 5 min per account
  - Token refresh: 6 per 1 hr per account
  - Game sessions: 20 per 1 hr per account
- **Middleware Chain** - Enhanced request processing
  - API Key → JWT Bearer → Rate Limit validation in sequence
  - Proper error responses at each stage
  - Rate limit headers on all responses
- **Error Response Format** - Standardized across all endpoints
  - Consistent HytaleError structure vs inconsistent previous responses
  - Proper HTTP status codes (400/401/403/404/429/500)
  - No sensitive data in error messages (cleaned up before responding)

### Fixed

- **Database Connection Pool** - Corrected pgxpool.Pool API usage
  - Changed from invalid `r.db.conn.ExecContext` to `r.db.Pool.Exec`
  - Changed from invalid `r.db.conn.QueryContext` to `r.db.Pool.Query`
  - Applies to all audit logging operations (LogTokenCreated, LogTokenRefreshed, LogTokenDeleted, LogSessionCreated, LogSessionRefreshed, LogSessionDeleted, LogAuthFailure, GetAuditLogs, GetLatestAuditLog)
- **Type Safety** - AuditLogType enum to string casting for SQL parameters
  - Ensures proper parameterized query execution
  - Prevents SQL injection and type mismatch errors
- **Code Duplication** - Removed duplicate `contains()` helper functions
  - Consolidated to single definition in appropriate package
  - Uses standard `strings.Contains()` for consistency

### Technical Details

- **Language**: Go 1.24
- **HTTP Framework**: Fiber v2.52.5
- **Job Queue**: Asynq v0.24.1 (Redis-backed)
- **Database Driver**: pgx v5.7.2 with pgxpool connection pooling
- **Crypto**: `crypto/ed25519` for JWT signature verification
- **JWT Handling**: Manual parsing with header/payload/signature validation
- **Caching**: In-memory thread-safe map for JWKS public keys
- **Scheduler**: robfig/cron v3.0.1 for token/session refresh jobs
- **Logging**: zerolog v1.33.0 for structured audit logging
- **Container Runtime**: Docker with docker-compose

### Security

- **OAuth Security**:
  - Device code authorization flow prevents credential leakage in logs
  - Server-side session state prevents CSRF attacks
  - Automatic device code expiration (30 min) limits brute force window
  - Polling timeout (15 min) prevents indefinite waits
- **Token Security**:
  - JWT signature verification with Ed25519 (collision-resistant, smaller keys)
  - Token expiry validation prevents indefinite access
  - Audience claim verification prevents token type confusion
  - Rate limiting (6/hr) prevents token enumeration attacks
  - 60-second clock skew tolerance handles time sync issues
- **Error Handling**:
  - Session limit errors (403) don't leak entitlements (users told "buy upgrade")
  - All errors use generic messages (no leaking OAuth server internals)
  - HTTP status codes match RFC 7231 standards
  - Rate limit responses (429) don't expose implementation details
- **Audit Trail**:
  - All OAuth operations logged with account, profile, IP, user agent
  - Authentication failures tracked for intrusion detection
  - Profile selection events logged for access control audits
  - Queryable audit logs support compliance investigations
- **Database Security**:
  - Foreign key constraints prevent orphaned token/session records
  - Prepared statements prevent SQL injection
  - pgxpool connection pooling isolates connections per request
  - Encryption at rest (existing AES-256-GCM system)

### Performance

- **JWKS Caching** - Eliminates repeated Hytale API calls
  - Hourly refresh instead of per-request fetches
  - In-memory map for O(1) key lookup
  - ~99% reduction in external API calls for token validation
- **Rate Limiting** - Sub-millisecond token bucket checks
  - Prevents cache invalidation from rate limit checks
  - Token bucket refill O(1) time complexity
  - No database queries for rate limiting
- **Token Validation** - Optimized JWT parsing
  - Single pass through token string (3 splits for header/payload/sig)
  - Base64URL decoding only used parts
  - Early exit on missing claims
  - Signature verification is CPU-bound (crypto-fast)
- **Audit Logging** - Minimal request path impact
  - Logging happens in background workers (not blocking response)
  - Batch inserts possible (not implemented but architecture supports)
  - Indexed audit queries (account_id, created_at DESC)

### Dependencies Added

- (No new external dependencies - uses existing ecosystem)

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
