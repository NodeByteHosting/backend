package types

// TokenValidationRequest represents a request to validate a token
type TokenValidationRequest struct {
	Token            string `json:"token" example:"eyJhbGciOiJFZERTQSJ9..."`
	TokenType        string `json:"token_type" example:"session"` // "session" or "identity"
	ExpectedAudience string `json:"expected_audience,omitempty" example:"hytale-server"`
}

// TokenValidationResponse represents the response from token validation
type TokenValidationResponse struct {
	Success   bool        `json:"success" example:"true"`
	Valid     bool        `json:"valid" example:"true"`
	Message   string      `json:"message,omitempty" example:"Token is valid and not expired"`
	Error     string      `json:"error,omitempty"`
	Claims    interface{} `json:"claims,omitempty"` // Decoded claims from token
	ExpiresAt string      `json:"expires_at,omitempty" example:"2024-01-13T16:30:00Z"`
}

// JWKSRefreshRequest represents a manual JWKS refresh request
type JWKSRefreshRequest struct {
	// No body required - just trigger refresh
}

// JWKSRefreshResponse represents the response from JWKS refresh
type JWKSRefreshResponse struct {
	Success     bool   `json:"success" example:"true"`
	KeysLoaded  int    `json:"keys_loaded" example:"3"`
	LastRefresh string `json:"last_refresh" example:"2024-01-13T15:30:00Z"`
	Message     string `json:"message,omitempty" example:"JWKS cache refreshed successfully"`
	Error       string `json:"error,omitempty"`
}

// JWKSStatusResponse represents the status of JWKS cache
type JWKSStatusResponse struct {
	Success     bool   `json:"success" example:"true"`
	CacheValid  bool   `json:"cache_valid" example:"true"`
	KeysCount   int    `json:"keys_count" example:"3"`
	LastRefresh string `json:"last_refresh" example:"2024-01-13T15:30:00Z"`
	NextRefresh string `json:"next_refresh" example:"2024-01-13T16:30:00Z"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}
