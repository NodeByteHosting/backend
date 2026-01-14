package types

// CredentialsRequest represents login credentials
type CredentialsRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"securepassword123"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Success bool      `json:"success" example:"true"`
	Token   string    `json:"token,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User    *UserData `json:"user,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// UserData represents user information in responses
type UserData struct {
	ID    string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email string `json:"email" example:"user@example.com"`
	Name  string `json:"name,omitempty" example:"John Doe"`
}

// RegisterUserRequest represents a user registration request
type RegisterUserRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"securepassword123"`
	Name     string `json:"name,omitempty" example:"John Doe"`
}

// VerifyEmailRequest represents an email verification request
type VerifyEmailRequest struct {
	Email string `json:"email" example:"user@example.com"`
	Code  string `json:"code" example:"123456"`
}

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Email       string `json:"email" example:"user@example.com"`
	Code        string `json:"code" example:"reset-token-123"`
	NewPassword string `json:"new_password" example:"newpassword123"`
}

// CheckEmailRequest represents an email existence check request
type CheckEmailRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

// CredentialsValidateRequest represents a credentials validation request
type CredentialsValidateRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"securepassword123"`
}

// MagicLinkRequest represents a magic link request
type MagicLinkRequest struct {
	Email string `json:"email" example:"user@example.com"`
}

// MagicLinkVerifyRequest represents a magic link verification request
type MagicLinkVerifyRequest struct {
	Email string `json:"email" example:"user@example.com"`
	Token string `json:"token" example:"magic-token-123"`
}
