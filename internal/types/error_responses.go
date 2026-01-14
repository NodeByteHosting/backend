package types

// DetailedErrorResponse represents a detailed error response with error code
type DetailedErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error" example:"Invalid request format or missing required fields"`
	Code    string `json:"code" example:"BAD_REQUEST"`
	Details string `json:"details,omitempty" example:"Missing required field: profile_uuid"`
}

// SessionLimitErrorResponse represents a session limit exceeded error
type SessionLimitErrorResponse struct {
	Success    bool   `json:"success" example:"false"`
	Error      string `json:"error" example:"Concurrent session limit reached (100 sessions). Account requires 'sessions.unlimited_servers' entitlement."`
	Code       string `json:"code" example:"SESSION_LIMIT_EXCEEDED"`
	StatusCode int    `json:"status_code" example:"403"`
}

// AuthErrorResponse represents an authentication error
type AuthErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error" example:"Missing or invalid authentication token"`
	Code    string `json:"code" example:"UNAUTHORIZED"`
}

// NotFoundErrorResponse represents a 404 error
type NotFoundErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error" example:"Resource not found (invalid profile UUID or account)"`
	Code    string `json:"code" example:"NOT_FOUND"`
}
