package types

// DeviceCodeRequest represents a device code request
type DeviceCodeRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// DeviceCodeResponseDTO represents the response from device code endpoint
type DeviceCodeResponseDTO struct {
	Success                 bool   `json:"success" example:"true"`
	DeviceCode              string `json:"device_code" example:"AH-wO0aD5nvS5xhd7rQw1qv6XUzC9Kk9IElVqxsqQ1KGIykN3iqjcqB5hFtMQxPuBs4uEA"`
	UserCode                string `json:"user_code" example:"GFHD-MJHT"`
	VerificationURI         string `json:"verification_uri" example:"https://accounts.hytale.com/oauth2/device?user_code=GFHD-MJHT"`
	VerificationURIComplete string `json:"verification_uri_complete" example:"https://accounts.hytale.com/oauth2/device?user_code=GFHD-MJHT"`
	ExpiresIn               int    `json:"expires_in" example:"900"`
	Interval                int    `json:"interval" example:"5"`
	Error                   string `json:"error,omitempty"`
}

// PollTokenRequest represents a token polling request
type PollTokenRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Device code from RequestDeviceCode response
	DeviceCode string `json:"device_code" example:"AH-wO0aD5nvS5xhd7rQw1qv6XUzC9Kk9IElVqxsqQ1KGIykN3iqjcqB5hFtMQxPuBs4uEA"`
}

// TokenResponseDTO represents an OAuth token response
type TokenResponseDTO struct {
	Success          bool   `json:"success" example:"true"`
	AccessToken      string `json:"access_token,omitempty" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken     string `json:"refresh_token,omitempty" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn        int    `json:"expires_in,omitempty" example:"3600"`
	TokenType        string `json:"token_type,omitempty" example:"Bearer"`
	Scope            string `json:"scope,omitempty" example:"openid offline auth:server"`
	Error            string `json:"error,omitempty" example:"authorization_pending"`
	ErrorDescription string `json:"error_description,omitempty" example:"The user has not yet completed the authorization process"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// GetProfilesRequest represents a get profiles request
type GetProfilesRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ProfileDTO represents a game profile
type ProfileDTO struct {
	// Profile UUID (game character UUID)
	UUID string `json:"uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	// Player username/character name
	Username string `json:"username" example:"PlayerName"`
}

// GetProfilesResponseDTO represents a get profiles response
type GetProfilesResponseDTO struct {
	Success  bool         `json:"success" example:"true"`
	Owner    string       `json:"owner,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Profiles []ProfileDTO `json:"profiles,omitempty"`
	Error    string       `json:"error,omitempty"`
}

// SelectProfileRequest represents a select profile request
type SelectProfileRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Profile/character UUID to select
	ProfileUUID string `json:"profile_uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// CreateGameSessionRequest represents a create game session request
type CreateGameSessionRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Profile/character UUID (optional if previously selected)
	ProfileUUID string `json:"profile_uuid,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// GameSessionDTO represents a game session
type GameSessionDTO struct {
	// Session token for authentication with game server
	SessionToken string `json:"session_token" example:"eyJhbGciOiJFZERTQSJ9..."`
	// Identity token containing player information
	IdentityToken string `json:"identity_token" example:"eyJhbGciOiJFZERTQSJ9..."`
	// Session expiry timestamp (1 hour from creation)
	ExpiresAt string `json:"expires_at" example:"2024-01-13T15:30:00Z"`
}

// CreateGameSessionResponseDTO represents a create game session response
type CreateGameSessionResponseDTO struct {
	Success bool           `json:"success" example:"true"`
	Session GameSessionDTO `json:"session,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// RefreshGameSessionRequest represents a refresh game session request
type RefreshGameSessionRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Profile/character UUID (optional if previously selected)
	ProfileUUID string `json:"profile_uuid,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// RefreshGameSessionResponseDTO represents a refresh game session response
type RefreshGameSessionResponseDTO struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message,omitempty" example:"Game session refreshed successfully"`
	Error   string `json:"error,omitempty"`
}

// TerminateGameSessionRequest represents a terminate game session request
type TerminateGameSessionRequest struct {
	// Account/Owner UUID from Hytale
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Profile/character UUID (optional if previously selected)
	ProfileUUID string `json:"profile_uuid,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
}

// TerminateGameSessionResponseDTO represents a terminate game session response
type TerminateGameSessionResponseDTO struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message,omitempty" example:"Game session terminated successfully"`
	Error   string `json:"error,omitempty"`
}
