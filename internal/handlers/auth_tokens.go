package handlers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/auth"
)

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// RefreshToken handles token refresh
// @Summary Refresh Access Token
// @Description Exchanges a valid refresh token for new access and refresh tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param refresh body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} AuthResponse "New tokens generated"
// @Failure 400 {object} AuthResponse "Missing refresh token"
// @Failure 401 {object} AuthResponse "Invalid or expired refresh token"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_refresh_token",
		})
	}

	// Validate refresh token from database
	session, err := h.db.GetSessionByToken(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_refresh_token",
		})
	}

	// Get user data
	user, err := h.db.QueryUserByID(c.Context(), session.UserID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "user_not_found",
		})
	}

	// Check if user is active
	if !user.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "account_disabled",
		})
	}

	// Generate new token pair
	claims := &auth.Claims{
		UserID:             user.ID,
		Email:              user.Email,
		Username:           user.Username.String,
		FirstName:          getStringPointer(user.FirstName),
		LastName:           getStringPointer(user.LastName),
		Roles:              user.Roles,
		IsPterodactylAdmin: user.IsPterodactylAdmin,
		IsVirtfusionAdmin:  user.IsVirtfusionAdmin,
		IsSystemAdmin:      user.IsSystemAdmin,
		PterodactylID:      getInt64Pointer(user.PterodactylID),
		EmailVerified:      formatNullTime(user.EmailVerified),
	}

	tokenPair, err := h.jwtService.GenerateTokenPair(claims)
	if err != nil {
		log.Error().Err(err).Str("userID", user.ID).Msg("Failed to generate tokens")
		return c.Status(fiber.StatusInternalServerError).JSON(AuthResponse{
			Success: false,
			Error:   "token_generation_failed",
		})
	}

	// Delete old refresh token
	_ = h.db.DeleteSession(c.Context(), req.RefreshToken)

	// Store new refresh token in session
	expiresAt := time.Now().Add(h.jwtService.GetRefreshTokenTTL())
	_, err = h.db.CreateSession(c.Context(), user.ID, tokenPair.RefreshToken, expiresAt)
	if err != nil {
		log.Error().Err(err).Str("userID", user.ID).Msg("Failed to create session")
		return c.Status(fiber.StatusInternalServerError).JSON(AuthResponse{
			Success: false,
			Error:   "session_creation_failed",
		})
	}

	log.Info().Str("userID", user.ID).Msg("Token refreshed")

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success:      true,
		Message:      "Token refreshed",
		Tokens:       tokenPair,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken,omitempty"`
}

// Logout handles user logout
// @Summary User Logout
// @Description Invalidates refresh token and terminates user session
// @Tags Authentication
// @Accept json
// @Produce json
// @Param logout body LogoutRequest false "Optional refresh token to invalidate"
// @Param Authorization header string false "Bearer token" example(Bearer eyJhbGc...)
// @Success 200 {object} AuthResponse "Logged out successfully"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req LogoutRequest
	_ = c.BodyParser(&req)

	// If refresh token provided, delete that specific session
	if req.RefreshToken != "" {
		err := h.db.DeleteSession(c.Context(), req.RefreshToken)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete session")
		}
	}

	// Also try to get user ID from JWT and delete all sessions
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := h.jwtService.ValidateAccessToken(token)
		if err == nil && claims != nil {
			// Delete all user sessions
			_ = h.db.DeleteUserSessions(c.Context(), claims.UserID)
			log.Info().Str("userID", claims.UserID).Msg("User logged out")
		}
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// GetCurrentUser returns the current authenticated user
// @Summary Get Current User
// @Description Returns authenticated user information from JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token" example(Bearer eyJhbGc...)
// @Success 200 {object} AuthResponse "User data"
// @Failure 401 {object} AuthResponse "Missing or invalid token"
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "missing_authorization",
		})
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtService.ValidateAccessToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_token",
		})
	}

	// Get fresh user data from database
	user, err := h.db.QueryUserByID(c.Context(), claims.UserID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "user_not_found",
		})
	}

	// Check if user is active
	if !user.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "account_disabled",
		})
	}

	userData := &UserData{
		ID:                 user.ID,
		Email:              user.Email,
		Username:           user.Username.String,
		FirstName:          getStringPointer(user.FirstName),
		LastName:           getStringPointer(user.LastName),
		Roles:              user.Roles,
		IsPterodactylAdmin: user.IsPterodactylAdmin,
		IsVirtfusionAdmin:  user.IsVirtfusionAdmin,
		IsSystemAdmin:      user.IsSystemAdmin,
		PterodactylID:      getInt64Pointer(user.PterodactylID),
		EmailVerified:      formatNullTime(user.EmailVerified),
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		User:    userData,
	})
}
