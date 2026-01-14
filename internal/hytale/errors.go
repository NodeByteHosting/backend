package hytale

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// HTTPErrorResponse represents an HTTP error response from Hytale APIs
type HTTPErrorResponse struct {
	StatusCode int
	Body       string
	RequestURL string
}

// HytaleError represents an application-level Hytale error
type HytaleError struct {
	Code                string // Machine-readable error code
	Message             string // User-friendly error message
	StatusCode          int    // HTTP status code to return
	InternalMsg         string // Internal details for logging
	IsSessionLimitError bool
}

// Error implements the error interface
func (e *HytaleError) Error() string {
	return e.Message
}

// NewHytaleError creates a new Hytale error
func NewHytaleError(code string, message string, statusCode int) *HytaleError {
	return &HytaleError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewHytaleErrorWithInternal creates a new Hytale error with internal details
func NewHytaleErrorWithInternal(code string, message string, statusCode int, internalMsg string) *HytaleError {
	return &HytaleError{
		Code:        code,
		Message:     message,
		StatusCode:  statusCode,
		InternalMsg: internalMsg,
	}
}

// SessionLimitError creates a 403 error for hitting concurrent session limit
func SessionLimitError() *HytaleError {
	return &HytaleError{
		Code:                "SESSION_LIMIT_EXCEEDED",
		Message:             "Concurrent session limit reached (100 sessions). Account requires 'sessions.unlimited_servers' entitlement.",
		StatusCode:          http.StatusForbidden,
		InternalMsg:         "Account hit 100 concurrent session limit",
		IsSessionLimitError: true,
	}
}

// HandleHTTPError converts HTTP error responses from Hytale APIs to HytaleError
func HandleHTTPError(statusCode int, body string, requestURL string) *HytaleError {
	switch statusCode {
	case http.StatusBadRequest:
		return &HytaleError{
			Code:        "BAD_REQUEST",
			Message:     "Invalid request format or missing required fields",
			StatusCode:  http.StatusBadRequest,
			InternalMsg: fmt.Sprintf("Bad request to %s: %s", requestURL, body),
		}

	case http.StatusUnauthorized:
		return &HytaleError{
			Code:        "UNAUTHORIZED",
			Message:     "Missing or invalid authentication token",
			StatusCode:  http.StatusUnauthorized,
			InternalMsg: fmt.Sprintf("Unauthorized request to %s: %s", requestURL, body),
		}

	case http.StatusForbidden:
		// Check if this is a session limit error
		if isSessionLimitError(body) {
			return SessionLimitError()
		}
		return &HytaleError{
			Code:        "FORBIDDEN",
			Message:     "Valid authentication but insufficient permissions",
			StatusCode:  http.StatusForbidden,
			InternalMsg: fmt.Sprintf("Forbidden request to %s: %s", requestURL, body),
		}

	case http.StatusNotFound:
		return &HytaleError{
			Code:        "NOT_FOUND",
			Message:     "Resource not found (invalid profile UUID or account)",
			StatusCode:  http.StatusNotFound,
			InternalMsg: fmt.Sprintf("Not found: %s - %s", requestURL, body),
		}

	case http.StatusTooManyRequests:
		return &HytaleError{
			Code:        "RATE_LIMITED",
			Message:     "Too many requests. Please retry after a delay.",
			StatusCode:  http.StatusTooManyRequests,
			InternalMsg: fmt.Sprintf("Rate limited on %s: %s", requestURL, body),
		}

	case http.StatusInternalServerError:
		return &HytaleError{
			Code:        "HYTALE_SERVER_ERROR",
			Message:     "Hytale service error. Please retry later.",
			StatusCode:  http.StatusBadGateway,
			InternalMsg: fmt.Sprintf("Hytale server error on %s: %s", requestURL, body),
		}

	case http.StatusServiceUnavailable:
		return &HytaleError{
			Code:        "SERVICE_UNAVAILABLE",
			Message:     "Hytale service is currently unavailable. Please retry later.",
			StatusCode:  http.StatusServiceUnavailable,
			InternalMsg: fmt.Sprintf("Service unavailable: %s", requestURL),
		}

	default:
		return &HytaleError{
			Code:        "HYTALE_ERROR",
			Message:     "An error occurred while communicating with Hytale",
			StatusCode:  http.StatusBadGateway,
			InternalMsg: fmt.Sprintf("HTTP %d from %s: %s", statusCode, requestURL, body),
		}
	}
}

// isSessionLimitError checks if the error response is a session limit error
func isSessionLimitError(body string) bool {
	// Check for common session limit error indicators in response
	indicators := []string{
		"session_limit",
		"limit_exceeded",
		"concurrent",
		"100",
		"unlimited_servers",
	}
	for _, indicator := range indicators {
		if strings.Contains(strings.ToLower(body), indicator) {
			return true
		}
	}
	return false
}

// LogError logs a HytaleError with appropriate context
func LogError(err *HytaleError, context map[string]string) {
	logger := log.Error().
		Str("code", err.Code).
		Str("message", err.Message).
		Int("status_code", err.StatusCode)

	if err.InternalMsg != "" {
		logger = logger.Str("internal", err.InternalMsg)
	}

	if err.IsSessionLimitError {
		logger = logger.Bool("session_limit", true)
	}

	// Add context fields
	for key, value := range context {
		logger = logger.Str(key, value)
	}

	logger.Msg("Hytale error occurred")
}
