package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/hytale"
	"github.com/nodebyte/backend/internal/types"
)

// HytaleOAuthHandler handles Hytale OAuth-related requests
type HytaleOAuthHandler struct {
	db          *database.DB
	oauthRepo   *database.HytaleOAuthRepository
	oauthClient *hytale.OAuthClient
}

// NewHytaleOAuthHandler creates a new Hytale OAuth handler
func NewHytaleOAuthHandler(db *database.DB, useStaging bool) *HytaleOAuthHandler {
	oauthClient := hytale.NewOAuthClient(&hytale.OAuthClientConfig{
		ClientID:   "hytale-server",
		UseStaging: useStaging,
	})

	return &HytaleOAuthHandler{
		db:          db,
		oauthRepo:   database.NewHytaleOAuthRepository(db),
		oauthClient: oauthClient,
	}
}

// RequestDeviceCode initiates device code flow
// @Summary Request Device Code
// @Description Initiates OAuth 2.0 Device Code Flow for Hytale server authentication
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.DeviceCodeRequest true "Device code request"
// @Success 200 {object} types.DeviceCodeResponseDTO
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/device-code [post]
func (h *HytaleOAuthHandler) RequestDeviceCode(c *fiber.Ctx) error {
	var req types.DeviceCodeRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid device code request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.AccountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	// Request device code from Hytale
	deviceResp, err := h.oauthClient.RequestDeviceCode(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to request device code from Hytale")
		return c.Status(http.StatusInternalServerError).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to request device code",
		})
	}

	log.Info().
		Str("account_id", req.AccountID).
		Str("user_code", deviceResp.UserCode).
		Msg("Device code requested")

	return c.JSON(types.DeviceCodeResponseDTO{
		Success:                 true,
		DeviceCode:              deviceResp.DeviceCode,
		UserCode:                deviceResp.UserCode,
		VerificationURI:         deviceResp.VerificationURI,
		VerificationURIComplete: deviceResp.VerificationURIComplete,
		ExpiresIn:               deviceResp.ExpiresIn,
		Interval:                deviceResp.Interval,
	})
}

// PollToken polls for token after user authorization
// @Summary Poll for Token
// @Description Polls Hytale OAuth endpoint to obtain access token after user authorization
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.PollTokenRequest true "Token polling request"
// @Success 200 {object} types.TokenResponseDTO
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/token [post]
func (h *HytaleOAuthHandler) PollToken(c *fiber.Ctx) error {
	var req types.PollTokenRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid token request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.DeviceCode == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "device_code is required",
		})
	}

	if req.AccountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	// Poll Hytale for token
	tokenResp, err := h.oauthClient.PollToken(c.Context(), req.DeviceCode)
	if err != nil {
		log.Error().Err(err).Msg("Failed to poll token from Hytale")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to poll token",
		})
	}

	// Check for authorization_pending error
	if tokenResp.Error == "authorization_pending" {
		return c.Status(http.StatusAccepted).JSON(types.TokenResponseDTO{
			Success:          false,
			Error:            "authorization_pending",
			ErrorDescription: "User has not yet authorized the device",
		})
	}

	// Check for other errors
	if tokenResp.Error != "" {
		log.Warn().
			Str("error", tokenResp.Error).
			Str("account_id", req.AccountID).
			Msg("Token request failed")
		return c.Status(http.StatusBadRequest).JSON(types.TokenResponseDTO{
			Success:          false,
			Error:            tokenResp.Error,
			ErrorDescription: tokenResp.ErrorDescription,
		})
	}

	// Save token to database
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	oauthToken := &database.HytaleOAuthToken{
		AccountID:         req.AccountID,
		AccessToken:       tokenResp.AccessToken,
		RefreshToken:      tokenResp.RefreshToken,
		AccessTokenExpiry: expiresAt,
		Scope:             tokenResp.Scope,
	}

	if err := h.oauthRepo.SaveOAuthToken(c.Context(), oauthToken); err != nil {
		log.Error().Err(err).Str("account_id", req.AccountID).Msg("Failed to save OAuth token")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to save token",
		})
	}

	log.Info().
		Str("account_id", req.AccountID).
		Msg("OAuth token obtained and stored")

	return c.JSON(types.TokenResponseDTO{
		Success:      true,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	})
}

// RefreshAccessToken refreshes an access token
// @Summary Refresh Access Token
// @Description Exchanges refresh token for a new access token
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} types.TokenResponseDTO
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "Token not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/refresh [post]
func (h *HytaleOAuthHandler) RefreshAccessToken(c *fiber.Ctx) error {
	var req types.RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid refresh token request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.AccountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	// Get stored token
	storedToken, err := h.oauthRepo.GetOAuthToken(c.Context(), req.AccountID)
	if err != nil {
		log.Warn().Err(err).Str("account_id", req.AccountID).Msg("Token not found")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "No token found for account",
		})
	}

	// Refresh token with Hytale
	tokenResp, err := h.oauthClient.RefreshToken(c.Context(), storedToken.RefreshToken)
	if err != nil {
		log.Error().Err(err).Str("account_id", req.AccountID).Msg("Failed to refresh token")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to refresh token",
		})
	}

	// Check for errors
	if tokenResp.Error != "" {
		log.Warn().
			Str("error", tokenResp.Error).
			Str("account_id", req.AccountID).
			Msg("Token refresh failed")
		return c.Status(http.StatusBadRequest).JSON(types.TokenResponseDTO{
			Success:          false,
			Error:            tokenResp.Error,
			ErrorDescription: tokenResp.ErrorDescription,
		})
	}

	// Update token in database
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	storedToken.AccessToken = tokenResp.AccessToken
	storedToken.RefreshToken = tokenResp.RefreshToken
	storedToken.AccessTokenExpiry = expiresAt
	storedToken.Scope = tokenResp.Scope

	if err := h.oauthRepo.SaveOAuthToken(c.Context(), storedToken); err != nil {
		log.Error().Err(err).Str("account_id", req.AccountID).Msg("Failed to update OAuth token")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to update token",
		})
	}

	log.Info().Str("account_id", req.AccountID).Msg("OAuth token refreshed")

	return c.JSON(types.TokenResponseDTO{
		Success:      true,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	})
}

// GetProfiles retrieves available game profiles
// @Summary Get Game Profiles
// @Description Fetches available game profiles for the authenticated account
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.GetProfilesRequest true "Get profiles request"
// @Success 200 {object} types.GetProfilesResponseDTO
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "Token not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/profiles [post]
func (h *HytaleOAuthHandler) GetProfiles(c *fiber.Ctx) error {
	var req types.GetProfilesRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid profiles request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.AccountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	// Get stored token
	storedToken, err := h.oauthRepo.GetOAuthToken(c.Context(), req.AccountID)
	if err != nil {
		log.Warn().Err(err).Str("account_id", req.AccountID).Msg("Token not found")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "No token found for account",
		})
	}

	// Fetch profiles from Hytale
	profileResp, err := h.oauthClient.GetProfiles(c.Context(), storedToken.AccessToken)
	if err != nil {
		log.Error().Err(err).Str("account_id", req.AccountID).Msg("Failed to get profiles")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch profiles",
		})
	}

	// Convert profiles
	profiles := make([]types.ProfileDTO, len(profileResp.Profiles))
	for i, p := range profileResp.Profiles {
		profiles[i] = types.ProfileDTO{
			UUID:     p.UUID,
			Username: p.Username,
		}
	}

	log.Info().
		Str("account_id", req.AccountID).
		Int("profile_count", len(profiles)).
		Msg("Profiles retrieved")

	return c.JSON(types.GetProfilesResponseDTO{
		Success:  true,
		Owner:    profileResp.Owner,
		Profiles: profiles,
	})
}

// SelectProfile selects a game profile
// @Summary Select Game Profile
// @Description Selects a specific game profile for token operations
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.SelectProfileRequest true "Select profile request"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/select-profile [post]
func (h *HytaleOAuthHandler) SelectProfile(c *fiber.Ctx) error {
	var req types.SelectProfileRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid select profile request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.AccountID == "" || req.ProfileUUID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id and profile_uuid are required",
		})
	}

	if err := h.oauthRepo.UpdateProfileUUID(c.Context(), req.AccountID, req.ProfileUUID); err != nil {
		log.Error().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", req.ProfileUUID).
			Msg("Failed to update profile")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to update profile selection",
		})
	}

	log.Info().
		Str("account_id", req.AccountID).
		Str("profile_uuid", req.ProfileUUID).
		Msg("Profile selected")

	return c.JSON(types.SuccessResponse{
		Success: true,
		Message: fmt.Sprintf("Profile %s selected", req.ProfileUUID),
	})
}

// CreateGameSession creates a new game session
// @Summary Create Game Session
// @Description Creates a new game session for the selected profile
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.CreateGameSessionRequest true "Create game session request"
// @Success 200 {object} types.CreateGameSessionResponseDTO
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "Token or profile not found"
// @Failure 403 {object} types.ErrorResponse "Session limit reached"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/game-session/new [post]
func (h *HytaleOAuthHandler) CreateGameSession(c *fiber.Ctx) error {
	var req types.CreateGameSessionRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid create game session request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.AccountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	// Get stored token
	storedToken, err := h.oauthRepo.GetOAuthToken(c.Context(), req.AccountID)
	if err != nil {
		log.Warn().Err(err).Str("account_id", req.AccountID).Msg("Token not found")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "No token found for account",
		})
	}

	// Use provided profile UUID or stored one
	profileUUID := req.ProfileUUID
	if profileUUID == "" {
		if storedToken.ProfileUUID.Valid {
			profileUUID = storedToken.ProfileUUID.String
		}
	}

	if profileUUID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "profile_uuid is required or must be selected first",
		})
	}

	// Create game session with Hytale
	sessionResp, err := h.oauthClient.CreateGameSession(c.Context(), storedToken.AccessToken, profileUUID)
	if err != nil {
		log.Error().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Failed to create game session")

		// Check if it's a session limit error
		if err.Error() == "hytale returned 403: " {
			return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
				Success: false,
				Error:   "Session limit reached for this account",
			})
		}

		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to create game session",
		})
	}

	// Save game session to database
	gameSession := &database.HytaleGameSession{
		AccountID:     req.AccountID,
		ProfileUUID:   profileUUID,
		SessionToken:  sessionResp.SessionToken,
		IdentityToken: sessionResp.IdentityToken,
	}

	if err := h.oauthRepo.SaveGameSession(c.Context(), gameSession); err != nil {
		log.Error().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Failed to save game session")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to save game session",
		})
	}

	log.Info().
		Str("account_id", req.AccountID).
		Str("profile_uuid", profileUUID).
		Msg("Game session created")

	return c.JSON(types.CreateGameSessionResponseDTO{
		Success: true,
		Session: types.GameSessionDTO{
			SessionToken:  sessionResp.SessionToken,
			IdentityToken: sessionResp.IdentityToken,
			ExpiresAt:     sessionResp.ExpiresAt,
		},
	})
}

// RefreshGameSession refreshes an existing game session
// @Summary Refresh Game Session
// @Description Refreshes an existing game session to extend its lifetime
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.RefreshGameSessionRequest true "Refresh game session request"
// @Success 200 {object} types.RefreshGameSessionResponseDTO
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "Session not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/game-session/refresh [post]
func (h *HytaleOAuthHandler) RefreshGameSession(c *fiber.Ctx) error {
	var req types.RefreshGameSessionRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid refresh game session request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.AccountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	// Get game session
	profileUUID := req.ProfileUUID
	if profileUUID == "" {
		// Get stored profile UUID
		token, err := h.oauthRepo.GetOAuthToken(c.Context(), req.AccountID)
		if err != nil {
			log.Warn().Err(err).Str("account_id", req.AccountID).Msg("Token not found")
			return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
				Success: false,
				Error:   "No token found for account",
			})
		}
		if token.ProfileUUID.Valid {
			profileUUID = token.ProfileUUID.String
		}
	}

	if profileUUID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "profile_uuid is required or must be selected first",
		})
	}

	gameSession, err := h.oauthRepo.GetGameSession(c.Context(), req.AccountID, profileUUID)
	if err != nil {
		log.Warn().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Game session not found")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "No game session found for account and profile",
		})
	}

	// Refresh session with Hytale
	sessionResp, err := h.oauthClient.RefreshGameSession(c.Context(), gameSession.SessionToken)
	if err != nil {
		log.Error().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Failed to refresh game session")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to refresh game session",
		})
	}

	if err := h.oauthRepo.UpdateGameSessionTokens(c.Context(), req.AccountID, profileUUID, sessionResp.SessionToken, sessionResp.IdentityToken); err != nil {
		log.Error().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Failed to update game session tokens")
		return c.Status(http.StatusInternalServerError).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to update session tokens",
		})
	}

	log.Info().
		Str("account_id", req.AccountID).
		Str("profile_uuid", profileUUID).
		Msg("Game session refreshed")

	return c.JSON(types.RefreshGameSessionResponseDTO{
		Success: true,
		Message: "Game session refreshed successfully",
	})
}

// TerminateGameSession terminates a game session
// @Summary Terminate Game Session
// @Description Terminates an active game session
// @Tags Hytale OAuth
// @Accept json
// @Produce json
// @Param payload body types.TerminateGameSessionRequest true "Terminate game session request"
// @Success 200 {object} types.TerminateGameSessionResponseDTO
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "Session not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/oauth/game-session/delete [post]
func (h *HytaleOAuthHandler) TerminateGameSession(c *fiber.Ctx) error {
	var req types.TerminateGameSessionRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid terminate game session request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.AccountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	// Get game session
	profileUUID := req.ProfileUUID
	if profileUUID == "" {
		// Get stored profile UUID
		token, err := h.oauthRepo.GetOAuthToken(c.Context(), req.AccountID)
		if err != nil {
			log.Warn().Err(err).Str("account_id", req.AccountID).Msg("Token not found")
			return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
				Success: false,
				Error:   "No token found for account",
			})
		}
		if token.ProfileUUID.Valid {
			profileUUID = token.ProfileUUID.String
		}
	}

	if profileUUID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "profile_uuid is required or must be selected first",
		})
	}

	gameSession, err := h.oauthRepo.GetGameSession(c.Context(), req.AccountID, profileUUID)
	if err != nil {
		log.Warn().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Game session not found")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "No game session found for account and profile",
		})
	}

	// Terminate session with Hytale
	if err := h.oauthClient.TerminateGameSession(c.Context(), gameSession.SessionToken); err != nil {
		log.Error().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Failed to terminate game session")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to terminate game session",
		})
	}

	// Delete session from database
	if err := h.oauthRepo.DeleteGameSession(c.Context(), req.AccountID, profileUUID); err != nil {
		log.Error().Err(err).
			Str("account_id", req.AccountID).
			Str("profile_uuid", profileUUID).
			Msg("Failed to delete game session from database")
		// Don't fail the response since the session was already terminated with Hytale
	}

	log.Info().
		Str("account_id", req.AccountID).
		Str("profile_uuid", profileUUID).
		Msg("Game session terminated")

	return c.JSON(types.TerminateGameSessionResponseDTO{
		Success: true,
		Message: "Game session terminated successfully",
	})
}
