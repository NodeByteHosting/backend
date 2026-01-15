package types

import "time"

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
	// Response data (generic for various endpoints)
	Data interface{} `json:"data,omitempty"`
}

// ServerLog represents a single server log entry
type ServerLog struct {
	// Log entry ID
	ID int64 `json:"id" example:"1"`
	// Log line content
	Line string `json:"line" example:"[10:30:45] Server started successfully"`
	// When the log was created
	Timestamp time.Time `json:"timestamp" example:"2026-01-14T10:30:45Z"`
}

// LogEntry represents a log line in a creation request
type LogEntry struct {
	// Log line content
	Line string `json:"line" example:"[10:30:45] Server started successfully"`
	// Optional timestamp (defaults to now)
	Timestamp time.Time `json:"timestamp,omitempty" example:"2026-01-14T10:30:45Z"`
}

// CreateServerLogsRequest represents a request to store server logs
type CreateServerLogsRequest struct {
	// Server UUID (for Hytale servers, this is the profile UUID)
	ServerUUID string `json:"server_uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	// Account ID of the server owner
	AccountID string `json:"account_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Array of log lines to store
	Logs []LogEntry `json:"logs"`
}
