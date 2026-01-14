## Overview

This API provides complete OAuth 2.0 Device Code Flow (RFC 8628) authentication and game session management for Hytale servers. It handles token lifecycle management, automatic refresh, and session tracking.

## Authentication Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Request Device Code → Get user_code & verification_uri  │
├─────────────────────────────────────────────────────────────┤
│ 2. User enters code at verification_uri on web browser      │
├─────────────────────────────────────────────────────────────┤
│ 3. Poll token endpoint → Receive access_token & refresh_token
├─────────────────────────────────────────────────────────────┤
│ 4. Get user profiles using access_token                     │
├─────────────────────────────────────────────────────────────┤
│ 5. Select profile → Bind to game session                    │
├─────────────────────────────────────────────────────────────┤
│ 6. Create game session → Receive session tokens             │
├─────────────────────────────────────────────────────────────┤
│ (Automatic) Refresh tokens 5 min before expiry              │
└─────────────────────────────────────────────────────────────┘
```

## Endpoints

### 1. Request Device Code

**Endpoint:** `POST /api/v1/hytale/oauth/device-code`

Initiates OAuth device code flow. Returns a device code and verification URI for user browser authentication.

**Rate Limit:** 5 requests per 15 minutes (per IP address)

**Request Body:**
```json
{}
```

**Success Response (200):**
```json
{
  "device_code": "DE123456789ABCDEF",
  "user_code": "AB12-CD34",
  "verification_uri": "https://accounts.hytale.com/device",
  "expires_in": 1800,
  "interval": 5
}
```

**Error Responses:**

- **400 Bad Request** - Invalid request format
  ```json
  {
    "code": "INVALID_REQUEST",
    "message": "Device code request failed",
    "status": 400
  }
  ```

- **429 Too Many Requests** - Rate limit exceeded
  ```json
  {
    "code": "RATE_LIMITED",
    "message": "Too many requests. Please try again later.",
    "status": 429,
    "headers": {
      "X-RateLimit-Limit": "5",
      "X-RateLimit-Remaining": "0",
      "X-RateLimit-Reset": "1705270800"
    }
  }
  ```

**Flow Instructions:**
1. Send request to get device code
2. Display `user_code` to user (format: XX00-XX00)
3. Instruct user to visit `verification_uri` and enter the code
4. Proceed to polling (endpoint #2)

---

### 2. Poll for Token

**Endpoint:** `POST /api/v1/hytale/oauth/token`

Polls Hytale OAuth server for authorization completion. Returns tokens once user authorizes.

**Rate Limit:** 10 requests per 5 minutes (per account ID)

**Request Body:**
```json
{
  "device_code": "DE123456789ABCDEF"
}
```

**Success Response (200):**
```json
{
  "access_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "refresh_eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "account_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Error Responses:**

- **400 Bad Request** - Invalid device code or missing fields
  ```json
  {
    "code": "INVALID_DEVICE_CODE",
    "message": "Device code not found or expired",
    "status": 400
  }
  ```

- **401 Unauthorized** - Device code still pending user authorization
  ```json
  {
    "code": "AUTHORIZATION_PENDING",
    "message": "Awaiting user authorization. Please try again in 5 seconds.",
    "status": 401,
    "retry_after": 5
  }
  ```

- **403 Forbidden** - Session limit reached (no premium entitlement)
  ```json
  {
    "code": "SESSION_LIMIT_EXCEEDED",
    "message": "Account has reached concurrent session limit (100). Upgrade to unlimited_servers to remove this restriction.",
    "status": 403,
    "entitlement_required": "sessions.unlimited_servers"
  }
  ```

- **404 Not Found** - Device code expired or invalid
  ```json
  {
    "code": "SESSION_NOT_FOUND",
    "message": "Device code expired (30 min limit)",
    "status": 404
  }
  ```

**Polling Strategy:**
- Implement exponential backoff: start with 5s, increase by 1s each attempt (max 15s)
- Stop polling after 15 minutes (timeout)
- Handle 401 responses by retrying after `retry_after` seconds
- On 403 SESSION_LIMIT, inform user they need premium entitlement

---

### 3. Refresh Access Token

**Endpoint:** `POST /api/v1/hytale/oauth/refresh`

Refreshes an expired or expiring access token using the refresh token (valid for 30 days).

**Rate Limit:** 6 requests per 1 hour (per account ID)

**Request Body:**
```json
{
  "refresh_token": "refresh_eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9..."
}
```

**Success Response (200):**
```json
{
  "access_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "refresh_eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

**Error Responses:**

- **401 Unauthorized** - Invalid or expired refresh token
  ```json
  {
    "code": "UNAUTHORIZED",
    "message": "Refresh token invalid or expired. Re-authenticate required.",
    "status": 401
  }
  ```

- **429 Too Many Requests** - Exceeded 6 refreshes per hour
  ```json
  {
    "code": "RATE_LIMITED",
    "message": "Token refresh limit exceeded",
    "status": 429
  }
  ```

**Notes:**
- This endpoint is automatically called by backend (every 5 minutes for OAuth tokens, every 10 minutes for game sessions)
- New refresh_token received on each successful refresh (old token invalidated)
- Keep refresh tokens secure - they grant 30-day access

---

### 4. Get User Profiles

**Endpoint:** `POST /api/v1/hytale/oauth/profiles`

Retrieves all game profiles associated with the authenticated account.

**Rate Limit:** 20 requests per 1 hour (per account ID)

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{}
```

**Success Response (200):**
```json
{
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "profiles": [
    {
      "uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "username": "PlayerName",
      "created_at": "2025-01-01T00:00:00Z"
    },
    {
      "uuid": "550e8400-e29b-41d4-a716-446655440001",
      "username": "AltCharacter",
      "created_at": "2025-06-15T12:30:00Z"
    }
  ]
}
```

**Error Responses:**

- **401 Unauthorized** - Missing or invalid access token
  ```json
  {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired access token",
    "status": 401
  }
  ```

- **403 Forbidden** - Token is valid but lacks required scope
  ```json
  {
    "code": "FORBIDDEN",
    "message": "Token missing required scope: openid",
    "status": 403
  }
  ```

**Usage:**
- Call after successful token polling
- Display profiles to user for selection
- Proceed to endpoint #5 (select profile)

---

### 5. Select Profile

**Endpoint:** `POST /api/v1/hytale/oauth/select-profile`

Binds a profile to the current session. Required before creating game session.

**Rate Limit:** 20 requests per 1 hour (per account ID)

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "profile_uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
```

**Success Response (200):**
```json
{
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "profile_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "username": "PlayerName",
  "selected_at": "2025-01-14T10:30:00Z"
}
```

**Error Responses:**

- **400 Bad Request** - Invalid profile UUID format
  ```json
  {
    "code": "INVALID_REQUEST",
    "message": "profile_uuid must be a valid UUID",
    "status": 400
  }
  ```

- **401 Unauthorized** - Invalid access token
  ```json
  {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired access token",
    "status": 401
  }
  ```

- **404 Not Found** - Profile UUID doesn't belong to this account
  ```json
  {
    "code": "SESSION_NOT_FOUND",
    "message": "Profile not found for this account",
    "status": 404
  }
  ```

---

### 6. Create Game Session

**Endpoint:** `POST /api/v1/hytale/oauth/game-session/new`

Creates a new game session with session tokens. Valid for 1 hour from creation.

**Rate Limit:** 20 requests per 1 hour (per account ID)

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "profile_uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
```

**Success Response (200):**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440002",
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "profile_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "session_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "identity_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-01-14T11:30:00Z",
  "created_at": "2025-01-14T10:30:00Z"
}
```

**Token Contents:**
- `session_token`: Contains session_id, profile_id, expiry. Used to verify player identity on server.
- `identity_token`: Contains player email, username, and account info. Use for profile display.

**Error Responses:**

- **401 Unauthorized** - Invalid access token
  ```json
  {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired access token",
    "status": 401
  }
  ```

- **403 Forbidden** - Session limit exceeded
  ```json
  {
    "code": "SESSION_LIMIT_EXCEEDED",
    "message": "Account has reached concurrent session limit (100). Upgrade to unlimited_servers.",
    "status": 403
  }
  ```

- **404 Not Found** - Profile doesn't exist
  ```json
  {
    "code": "SESSION_NOT_FOUND",
    "message": "Profile not found",
    "status": 404
  }
  ```

**Notes:**
- Session tokens automatically refreshed every 10 minutes by backend
- Player can use `session_token` to authenticate with Hytale game servers
- Keep session tokens secret (equivalent to passwords)

---

### 7. Refresh Game Session

**Endpoint:** `POST /api/v1/hytale/oauth/game-session/refresh`

Refreshes an active game session to extend its 1-hour lifetime. Only works within last 10 minutes before expiry.

**Rate Limit:** 20 requests per 1 hour (per account ID)

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440002"
}
```

**Success Response (200):**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440002",
  "session_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "identity_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-01-14T12:30:00Z",
  "refreshed_at": "2025-01-14T10:40:00Z"
}
```

**Error Responses:**

- **400 Bad Request** - Session not yet eligible for refresh
  ```json
  {
    "code": "INVALID_REQUEST",
    "message": "Session cannot be refreshed until 10 minutes before expiry",
    "status": 400
  }
  ```

- **401 Unauthorized** - Invalid access token
  ```json
  {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired access token",
    "status": 401
  }
  ```

- **404 Not Found** - Session doesn't exist or is expired
  ```json
  {
    "code": "SESSION_NOT_FOUND",
    "message": "Session not found or expired",
    "status": 404
  }
  ```

**Notes:**
- Backend automatically calls this every 10 minutes
- Calling manually provides same result as automatic refresh
- Receiving new session_token invalidates previous token

---

### 8. Terminate Game Session

**Endpoint:** `POST /api/v1/hytale/oauth/game-session/delete`

Terminates an active game session. Call on server shutdown or logout.

**Rate Limit:** 20 requests per 1 hour (per account ID)

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440002"
}
```

**Success Response (200):**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440002",
  "terminated_at": "2025-01-14T10:45:00Z",
  "status": "deleted"
}
```

**Error Responses:**

- **404 Not Found** - Session doesn't exist
  ```json
  {
    "code": "SESSION_NOT_FOUND",
    "message": "Session not found",
    "status": 404
  }
  ```

**Notes:**
- Sessions auto-expire after 1 hour if not refreshed
- Calling this endpoint forcibly terminates the session immediately
- Use when player disconnects from server

---

## Token Validation

All `session_token` and `identity_token` are JWT (JSON Web Tokens) signed with Ed25519 keys.

### Signature Verification

1. Fetch JWKS (public keys) from Hytale endpoint (cached hourly)
2. Extract JWT header to find `kid` (key ID)
3. Retrieve public key by `kid` from JWKS
4. Verify signature using Ed25519 verification
5. Decode payload and validate claims

### Required Claims

**session_token:**
- `sub` - Profile UUID
- `aud` - Should contain "sessions"
- `exp` - Expiration timestamp (Unix)
- `iat` - Issued-at timestamp (Unix)
- `session_id` - Custom claim with session identifier

**identity_token:**
- `sub` - Account UUID
- `aud` - Should contain "identities"
- `exp` - Expiration timestamp (Unix)
- `email` - User email address
- `preferred_username` - Display name

### Validation Example (Go)

```go
import "github.com/nodebyte/backend/internal/hytale"

validator := hytale.NewTokenValidator(jwksCache)

// Validate session token
sessionClaims, err := validator.ValidateSessionToken(sessionTokenString)
if err != nil {
    // Handle invalid token
    log.Printf("Invalid session token: %v", err)
    return
}

profileID := sessionClaims.Sub // Use as player identifier
```

---

## Error Codes Reference

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| INVALID_REQUEST | 400 | Invalid request format/parameters | No |
| INVALID_DEVICE_CODE | 400 | Device code invalid or expired | No - get new code |
| UNAUTHORIZED | 401 | Missing/invalid access token | Yes - refresh token |
| AUTHORIZATION_PENDING | 401 | User hasn't authorized yet | Yes - follow retry_after |
| FORBIDDEN | 403 | Valid token but insufficient permissions | No |
| SESSION_LIMIT_EXCEEDED | 403 | 100 concurrent session limit | No - user needs upgrade |
| SESSION_NOT_FOUND | 404 | Session/profile/code doesn't exist | No |
| ENDPOINT_NOT_FOUND | 404 | Invalid endpoint URL | No |
| RATE_LIMITED | 429 | Too many requests in time window | Yes - after X-RateLimit-Reset |
| SERVICE_ERROR | 500+ | Internal server error | Yes - with exponential backoff |

---

## Rate Limiting

All endpoints return rate limit headers:

```
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 3
X-RateLimit-Reset: 1705270800
```

- `X-RateLimit-Limit` - Maximum requests in window
- `X-RateLimit-Remaining` - Requests left
- `X-RateLimit-Reset` - Unix timestamp when limit resets

**Per-Endpoint Limits:**
- Device Code: 5/15min (per IP)
- Token Poll: 10/5min (per account)
- Profiles: 20/hour (per account)
- Game Session: 20/hour (per account)

---

## Audit Logging

All OAuth operations are logged for compliance:

- Token creation (timestamp, account, IP, user agent)
- Token refresh (when, which token, success/failure)
- Session operations (created/refreshed/deleted, when)
- Auth failures (reason, IP, timestamp)

Logs accessible via admin dashboard for forensics and compliance investigations.

---

## Best Practices

1. **Token Storage**
   - Store tokens in secure, encrypted database
   - Never log tokens in plaintext
   - Use HTTPS for all API calls

2. **Refresh Strategy**
   - Refresh tokens 5-10 minutes before expiry
   - Implement exponential backoff on refresh failures
   - Handle 401 by requesting new device code

3. **Session Management**
   - Create new session on each server login
   - Terminate session on logout/server shutdown
   - Handle 403 SESSION_LIMIT gracefully (show upgrade prompt)

4. **Error Handling**
   - Implement retry logic with backoff for 401, 429, 5xx
   - Never retry on 400, 403, 404
   - Display user-friendly error messages (not technical details)

5. **Security**
   - Validate JWT signatures before trusting token claims
   - Use HTTPS (never HTTP)
   - Implement CSRF tokens if storing access tokens in cookies
   - Never expose tokens in URLs or logs

