package handlers

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// APIKeyMiddleware handles X-API-Key authentication
type APIKeyMiddleware struct {
	apiKey string
}

// NewAPIKeyMiddleware creates a new API key middleware
func NewAPIKeyMiddleware(apiKey string) *APIKeyMiddleware {
	return &APIKeyMiddleware{apiKey: apiKey}
}

// Handler returns the middleware handler
func (m *APIKeyMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" || apiKey != m.apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "Invalid or missing API key",
				Code:    "UNAUTHORIZED",
			})
		}

		return c.Next()
	}
}

// BearerAuthMiddleware handles JWT Bearer token authentication
type BearerAuthMiddleware struct {
	db *database.DB
}

// NewBearerAuthMiddleware creates a new Bearer auth middleware
func NewBearerAuthMiddleware(db *database.DB) *BearerAuthMiddleware {
	return &BearerAuthMiddleware{db: db}
}

// Handler returns the middleware handler function
func (m *BearerAuthMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Error().Msg("Missing Authorization header")
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "Missing Authorization header",
				Code:    "UNAUTHORIZED",
			})
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "Invalid Authorization header format",
				Code:    "UNAUTHORIZED",
			})
		}

		token := parts[1]

		// Decode JWT payload (without signature verification - we'll validate in DB)
		// JWT format: header.payload.signature
		tokenParts := strings.Split(token, ".")
		if len(tokenParts) != 3 {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "Invalid token format",
				Code:    "UNAUTHORIZED",
			})
		}

		// Decode the payload (second part) using base64url
		payload := tokenParts[1]
		// Add padding if needed (JWT uses base64url)
		payload += strings.Repeat("=", (4-len(payload)%4)%4)

		decodedPayload, err := base64.RawURLEncoding.DecodeString(payload)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "Invalid token",
				Code:    "UNAUTHORIZED",
			})
		}

		// Parse JSON payload
		var claims map[string]interface{}
		err = json.Unmarshal(decodedPayload, &claims)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "Invalid token claims",
				Code:    "UNAUTHORIZED",
			})
		}

		// Get user ID from claims
		userID, ok := claims["id"].(string)
		if !ok || userID == "" {
			log.Error().
				Interface("claims", claims).
				Msg("Invalid token: failed to extract user ID from claims")
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "Invalid token: missing user ID",
				Code:    "UNAUTHORIZED",
			})
		}

		// Query database to verify user exists and is admin
		var isSystemAdmin bool
		err = m.db.Pool.QueryRow(c.Context(),
			"SELECT \"isSystemAdmin\" FROM users WHERE id = $1 LIMIT 1",
			userID,
		).Scan(&isSystemAdmin)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("User not found in database or query error")
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Success: false,
				Error:   "User not found",
				Code:    "UNAUTHORIZED",
			})
		}

		if !isSystemAdmin {
			log.Warn().Str("user_id", userID).Msg("Non-admin user attempted admin access")
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Success: false,
				Error:   "Admin access required",
				Code:    "FORBIDDEN",
			})
		}

		// Store user ID in context for handlers
		c.Locals("userID", userID)
		c.Locals("isAdmin", true)

		return c.Next()
	}
}
