package handlers

import (
	"database/sql"
	"errors"
	"regexp"
	"time"
	"unicode"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/auth"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/queue"
)

// AuthHandler handles authentication-related API requests
type AuthHandler struct {
	db           *database.DB
	queueManager *queue.Manager
	jwtService   *auth.JWTService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *database.DB, queueManager *queue.Manager, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		db:           db,
		queueManager: queueManager,
		jwtService:   jwtService,
	}
}

// CredentialsRequest represents a credentials authentication request
type CredentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Success      bool            `json:"success"`
	Message      string          `json:"message,omitempty"`
	Error        string          `json:"error,omitempty"`
	User         *UserData       `json:"user,omitempty"`
	Token        string          `json:"token,omitempty"` // Deprecated: use tokens instead
	Tokens       *auth.TokenPair `json:"tokens,omitempty"`
	AccessToken  string          `json:"accessToken,omitempty"`  // For backward compatibility
	RefreshToken string          `json:"refreshToken,omitempty"` // For backward compatibility
	ExpiresIn    int64           `json:"expiresIn,omitempty"`    // For backward compatibility
}

// UserData represents user information returned during auth
type UserData struct {
	ID                 string   `json:"id"`
	Email              string   `json:"email"`
	Username           string   `json:"username"`
	FirstName          *string  `json:"firstName"`
	LastName           *string  `json:"lastName"`
	Roles              []string `json:"roles"`
	IsPterodactylAdmin bool     `json:"isPterodactylAdmin"`
	IsVirtfusionAdmin  bool     `json:"isVirtfusionAdmin"`
	IsSystemAdmin      bool     `json:"isSystemAdmin"`
	PterodactylID      *int     `json:"pterodactylId"`
	EmailVerified      *string  `json:"emailVerified"`
}

// ValidatePassword checks if password meets requirements
// Requires: minimum 8 characters, at least 1 uppercase, 1 lowercase, 1 digit
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password_too_short")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false

	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasUpper {
		return errors.New("password_needs_uppercase")
	}
	if !hasLower {
		return errors.New("password_needs_lowercase")
	}
	if !hasDigit {
		return errors.New("password_needs_digit")
	}

	return nil
}

// ValidateEmail checks if email is valid
func validateEmail(email string) error {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	if !matched {
		return errors.New("invalid_email")
	}
	return nil
}

// AuthenticateUser handles user login with credentials
// @Summary User Login
// @Description Authenticates a user with email and password, returns JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body CredentialsRequest true "Login credentials"
// @Success 200 {object} AuthResponse "Login successful with JWT tokens"
// @Failure 400 {object} AuthResponse "Invalid request"
// @Failure 401 {object} AuthResponse "Invalid credentials or email not verified"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) AuthenticateUser(c *fiber.Ctx) error {
	var req CredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_fields",
		})
	}

	if err := validateEmail(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_email",
		})
	}

	// Query database for user
	user, err := h.db.QueryUserByEmail(c.Context(), req.Email)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_credentials",
		})
	}

	// Verify password
	if !user.VerifyPassword(req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_credentials",
		})
	}

	// Check if email is verified
	if !user.EmailVerified.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "email_not_verified",
		})
	}

	// Generate JWT tokens
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

	// Store refresh token in session
	expiresAt := time.Now().Add(h.jwtService.GetRefreshTokenTTL())
	_, err = h.db.CreateSession(c.Context(), user.ID, tokenPair.RefreshToken, expiresAt)
	if err != nil {
		log.Error().Err(err).Str("userID", user.ID).Msg("Failed to create session")
		return c.Status(fiber.StatusInternalServerError).JSON(AuthResponse{
			Success: false,
			Error:   "session_creation_failed",
		})
	}

	// Return user data with tokens
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
		Success:      true,
		Message:      "Login successful",
		User:         userData,
		Tokens:       tokenPair,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// RegisterUserRequest represents a user registration request
type RegisterUserRequest struct {
	Email           string  `json:"email"`
	Password        string  `json:"password"`
	ConfirmPassword string  `json:"confirmPassword"`
	Username        *string `json:"username,omitempty"`
	FirstName       *string `json:"firstName,omitempty"`
	LastName        *string `json:"lastName,omitempty"`
}

// RegisterUser handles user registration
// @Summary User Registration
// @Description Registers a new user account and sends verification email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param registration body RegisterUserRequest true "Registration details"
// @Success 201 {object} AuthResponse "User registered successfully"
// @Failure 400 {object} AuthResponse "Invalid request or validation error"
// @Failure 409 {object} AuthResponse "Email already exists"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) RegisterUser(c *fiber.Ctx) error {
	var req RegisterUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_fields",
		})
	}

	if err := validateEmail(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_email",
		})
	}

	if err := validatePassword(req.Password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	if req.Password != req.ConfirmPassword {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "passwords_dont_match",
		})
	}

	// Check if user already exists
	existing, err := h.db.QueryUserByEmail(c.Context(), req.Email)
	if err == nil && existing != nil {
		return c.Status(fiber.StatusConflict).JSON(AuthResponse{
			Success: false,
			Error:   "email_exists",
		})
	}

	// Generate default username if not provided
	username := req.Username
	if username == nil || *username == "" {
		parts := req.Email
		for i, ch := range parts {
			if ch == '@' {
				parts = parts[:i]
				break
			}
		}
		username = &parts
	}

	// Create new user
	user, err := h.db.CreateUser(c.Context(), &database.User{
		Email:     req.Email,
		Username:  database.NewNullString(*username),
		FirstName: database.NewNullString(getPointerValue(req.FirstName)),
		LastName:  database.NewNullString(getPointerValue(req.LastName)),
		Roles:     []string{"MEMBER"},
	}, req.Password)

	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to create user")
		return c.Status(fiber.StatusInternalServerError).JSON(AuthResponse{
			Success: false,
			Error:   "server_error",
		})
	}

	// Generate verification token
	token, err := h.db.StoreVerificationToken(
		c.Context(),
		user.ID,
		database.VerificationTokenType,
		database.TokenExpiration,
	)

	if err != nil {
		log.Error().Err(err).Str("userID", user.ID).Msg("Failed to generate verification token")
		// Continue anyway - user can request new token via email
	}

	// Queue verification email
	if err == nil && h.queueManager != nil && token != "" {
		_, _ = h.queueManager.EnqueueEmail(queue.EmailPayload{
			To:       user.Email,
			Subject:  "Verify your email",
			Template: "verify-email",
			Data: map[string]string{
				"name":  getPointerValue(req.FirstName),
				"token": token,
				"email": user.Email,
			},
		})
	}

	log.Info().Str("email", req.Email).Str("userID", user.ID).Msg("User registered successfully")

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		Success: true,
		Message: "Registration successful. Please verify your email.",
	})
}

// VerifyEmailRequest represents an email verification request
type VerifyEmailRequest struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

// VerifyEmail handles email verification
// @Summary Verify Email
// @Description Verifies user email address with verification token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param verification body map[string]string true "Verification token" example({"token":"abc123"})
// @Success 200 {object} AuthResponse "Email verified successfully"
// @Failure 400 {object} AuthResponse "Invalid or missing token"
// @Failure 404 {object} AuthResponse "User not found"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	if req.Token == "" || req.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_fields",
		})
	}

	// Verify the token in database
	verified, err := h.db.VerifyEmailToken(c.Context(), req.ID, req.Token)
	if err != nil || !verified {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "Email verified successfully",
	})
}

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles forgot password requests
// @Summary Forgot Password
// @Description Sends password reset email with reset token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param email body ForgotPasswordRequest true "User email"
// @Success 200 {object} AuthResponse "Reset email sent"
// @Failure 400 {object} AuthResponse "Invalid email"
// @Failure 404 {object} AuthResponse "User not found"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_email",
		})
	}

	if err := validateEmail(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_email",
		})
	}

	// Check if user exists (but don't reveal if they do or don't)
	user, err := h.db.QueryUserByEmail(c.Context(), req.Email)
	if err == nil && user != nil {
		// Generate password reset token
		token, err := h.db.StoreVerificationToken(
			c.Context(),
			user.ID,
			database.PasswordResetTokenType,
			database.TokenExpiration,
		)

		if err != nil {
			log.Error().Err(err).Str("email", req.Email).Msg("Failed to generate reset token")
		} else if token != "" && h.queueManager != nil {
			// Queue password reset email
			_, _ = h.queueManager.EnqueueEmail(queue.EmailPayload{
				To:       user.Email,
				Subject:  "Reset your password",
				Template: "reset-password",
				Data: map[string]string{
					"name":  user.FirstName.String,
					"token": token,
					"email": user.Email,
				},
			})
			log.Info().Str("email", req.Email).Msg("Password reset requested")
		}
	}

	// Always return success for security
	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "If an account exists with that email, you will receive a password reset link",
	})
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Token           string `json:"token"`
	ID              string `json:"id"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

// ResetPassword handles password reset
// @Summary Reset Password
// @Description Resets user password using reset token from email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param reset body ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} AuthResponse "Password reset successfully"
// @Failure 400 {object} AuthResponse "Invalid request or weak password"
// @Failure 404 {object} AuthResponse "User not found or invalid token"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	if req.Token == "" || req.ID == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_fields",
		})
	}

	if req.Password != req.ConfirmPassword {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "passwords_dont_match",
		})
	}

	if err := validatePassword(req.Password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Verify reset token and update password
	success, err := h.db.ResetUserPassword(c.Context(), req.ID, req.Token, req.Password)
	if err != nil || !success {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "Password reset successfully",
	})
}

// GetUserByID retrieves user information by ID
// @Summary Get User By ID
// @Description Retrieves user information by user ID
// @Tags Authentication
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} AuthResponse "User information"
// @Failure 404 {object} AuthResponse "User not found"
// @Router /api/v1/auth/users/{id} [get]
func (h *AuthHandler) GetUserByID(c *fiber.Ctx) error {
	userID := c.Params("id")

	user, err := h.db.QueryUserByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(AuthResponse{
			Success: false,
			Error:   "user_not_found",
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

// Helper functions for type conversion

// formatNullTime converts sql.NullTime to string pointer
func formatNullTime(t sql.NullTime) *string {
	if t.Valid {
		s := t.Time.Format("2006-01-02T15:04:05Z07:00")
		return &s
	}
	return nil
}

// getStringPointer safely converts sql.NullString to *string
func getStringPointer(ns sql.NullString) *string {
	if ns.Valid && ns.String != "" {
		return &ns.String
	}
	return nil
}

// getInt64Pointer safely converts sql.NullInt64 to *int
func getInt64Pointer(ni sql.NullInt64) *int {
	if ni.Valid && ni.Int64 != 0 {
		i := int(ni.Int64)
		return &i
	}
	return nil
}

// getPointerValue safely extracts string value from pointer
func getPointerValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// CheckEmailExistsRequest represents a check email request
type CheckEmailRequest struct {
	Email string `json:"email"`
}

// CheckEmailExists checks if an email is already registered
// @Summary Check Email Exists
// @Description Checks if an email address is already registered in the system
// @Tags Authentication
// @Accept json
// @Produce json
// @Param email query string true "Email address to check"
// @Success 200 {object} map[string]interface{} "Email availability status"
// @Failure 400 {object} map[string]interface{} "Invalid email"
// @Router /api/v1/auth/check-email [get]
func (h *AuthHandler) CheckEmailExists(c *fiber.Ctx) error {
	email := c.Query("email")

	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "missing_email",
		})
	}

	if err := validateEmail(email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid_email",
		})
	}

	user, _ := h.db.QueryUserByEmail(c.Context(), email)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"exists":  user != nil,
	})
}

// CredentialsValidateRequest represents a credentials validation request (for NextAuth custom provider)
type CredentialsValidateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ValidateCredentials validates credentials and returns user data for NextAuth
// This is specifically designed for NextAuth custom provider integration
// @Summary Validate Credentials
// @Description Validates user credentials without creating a session
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body CredentialsValidateRequest true "Credentials to validate"
// @Success 200 {object} map[string]interface{} "Credentials are valid"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Router /api/v1/auth/validate [post]
func (h *AuthHandler) ValidateCredentials(c *fiber.Ctx) error {
	var req CredentialsValidateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid_request",
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "missing_fields",
		})
	}

	if err := validateEmail(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid_email",
		})
	}

	// Query database for user
	user, err := h.db.QueryUserByEmail(c.Context(), req.Email)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "invalid_credentials",
		})
	}

	// Verify password
	if !user.VerifyPassword(req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "invalid_credentials",
		})
	}

	// Check if email is verified
	if !user.EmailVerified.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "email_not_verified",
		})
	}

	// Update last login
	_ = h.db.UpdateLastLogin(c.Context(), user.ID)

	// Return user data formatted for NextAuth
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"user": fiber.Map{
			"id":                 user.ID,
			"email":              user.Email,
			"username":           user.Username.String,
			"firstName":          getStringPointer(user.FirstName),
			"lastName":           getStringPointer(user.LastName),
			"roles":              user.Roles,
			"isPterodactylAdmin": user.IsPterodactylAdmin,
			"isVirtfusionAdmin":  user.IsVirtfusionAdmin,
			"isSystemAdmin":      user.IsSystemAdmin,
			"pterodactylId":      getInt64Pointer(user.PterodactylID),
			"emailVerified":      user.EmailVerified.Time.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

// MagicLinkRequest represents a magic link request
type MagicLinkRequest struct {
	Email string `json:"email"`
}

// RequestMagicLink sends a magic link to the user's email
// @Summary Request Magic Link
// @Description Sends a passwordless authentication magic link to user's email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param magicLink body MagicLinkRequest true "User email"
// @Success 200 {object} AuthResponse "Magic link sent"
// @Failure 400 {object} AuthResponse "Invalid email"
// @Failure 404 {object} AuthResponse "User not found"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/magic-link [post]
func (h *AuthHandler) RequestMagicLink(c *fiber.Ctx) error {
	var req MagicLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_email",
		})
	}

	if err := validateEmail(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_email",
		})
	}

	// Check if user exists
	user, err := h.db.QueryUserByEmail(c.Context(), req.Email)

	// Always return success for security
	if user != nil && err == nil {
		// Generate magic link token (30 minute expiration)
		token, err := h.db.StoreVerificationToken(
			c.Context(),
			user.ID,
			database.MagicLinkTokenType,
			database.MagicLinkExpiration,
		)

		if err != nil {
			log.Error().Err(err).Str("email", req.Email).Msg("Failed to generate magic link token")
		} else if token != "" && h.queueManager != nil {
			// Queue magic link email
			_, _ = h.queueManager.EnqueueEmail(queue.EmailPayload{
				To:       user.Email,
				Subject:  "Your magic link",
				Template: "magic-link",
				Data: map[string]string{
					"name":  user.FirstName.String,
					"token": token,
					"email": user.Email,
				},
			})
			log.Info().Str("email", req.Email).Msg("Magic link requested")
		}
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "If an account exists with that email, a magic link will be sent",
	})
}

// MagicLinkVerifyRequest represents a magic link verification
type MagicLinkVerifyRequest struct {
	Token string `json:"token"`
}

// VerifyMagicLink verifies a magic link token
// @Summary Verify Magic Link
// @Description Verifies magic link token and authenticates user
// @Tags Authentication
// @Accept json
// @Produce json
// @Param verify body MagicLinkVerifyRequest true "Magic link token"
// @Success 200 {object} AuthResponse "Authentication successful with JWT tokens"
// @Failure 400 {object} AuthResponse "Invalid request"
// @Failure 401 {object} AuthResponse "Invalid or expired token"
// @Failure 500 {object} AuthResponse "Internal server error"
// @Router /api/v1/auth/magic-link/verify [post]
func (h *AuthHandler) VerifyMagicLink(c *fiber.Ctx) error {
	var req MagicLinkVerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_request",
		})
	}

	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Error:   "missing_token",
		})
	}

	// Consume magic link token (one-time use)
	userID, err := h.db.ConsumeMagicLinkToken(c.Context(), req.Token)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid or expired magic link")
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Error:   "invalid_token",
		})
	}

	// Fetch user data to return
	user, err := h.db.QueryUserByID(c.Context(), userID)
	if err != nil || user == nil {
		log.Error().Err(err).Str("userID", userID).Msg("Failed to fetch user after magic link verification")
		return c.Status(fiber.StatusInternalServerError).JSON(AuthResponse{
			Success: false,
			Error:   "server_error",
		})
	}

	// Update last login
	_ = h.db.UpdateLastLogin(c.Context(), userID)

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

	log.Info().Str("userID", userID).Msg("Magic link verified")

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "Magic link verified",
		User:    userData,
	})
}
