package types

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	// Whether the request was successful
	Success bool `json:"success" example:"false"`
	// Human-readable error message
	Error string `json:"error" example:"account_id is required"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	// Whether the request was successful
	Success bool `json:"success" example:"true"`
	// Optional message
	Message string `json:"message,omitempty" example:"Operation completed successfully"`
}
