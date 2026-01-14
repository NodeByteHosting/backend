package hytale

import (
	"net/http"
	"testing"
)

func TestNewHytaleError(t *testing.T) {
	err := NewHytaleError("TEST_CODE", "Test message", http.StatusBadRequest)

	if err == nil {
		t.Fatal("expected error to be created")
	}
	if err.Code != "TEST_CODE" {
		t.Errorf("expected code TEST_CODE, got %s", err.Code)
	}
	if err.Message != "Test message" {
		t.Errorf("expected message 'Test message', got %s", err.Message)
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, err.StatusCode)
	}
}

func TestNewHytaleErrorWithInternal(t *testing.T) {
	err := NewHytaleErrorWithInternal("INTERNAL_CODE", "User message", http.StatusInternalServerError, "Internal details")

	if err == nil {
		t.Fatal("expected error to be created")
	}
	if err.Code != "INTERNAL_CODE" {
		t.Errorf("expected code INTERNAL_CODE, got %s", err.Code)
	}
	if err.Message != "User message" {
		t.Errorf("expected message 'User message', got %s", err.Message)
	}
	if err.InternalMsg != "Internal details" {
		t.Errorf("expected internal message 'Internal details', got %s", err.InternalMsg)
	}
	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, err.StatusCode)
	}
}

func TestHytaleErrorInterface(t *testing.T) {
	err := NewHytaleError("CODE", "Message", http.StatusBadRequest)

	// Should implement error interface
	var _ error = err

	if err.Error() != "Message" {
		t.Errorf("expected Error() to return 'Message', got %s", err.Error())
	}
}

func TestSessionLimitError(t *testing.T) {
	err := SessionLimitError()

	if err == nil {
		t.Fatal("expected session limit error to be created")
	}
	if err.Code != "SESSION_LIMIT_EXCEEDED" {
		t.Errorf("expected code SESSION_LIMIT_EXCEEDED, got %s", err.Code)
	}
	if !err.IsSessionLimitError {
		t.Errorf("expected IsSessionLimitError to be true")
	}
	if err.StatusCode != http.StatusForbidden {
		t.Errorf("expected status code %d, got %d", http.StatusForbidden, err.StatusCode)
	}
}

func TestHandleHTTPError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "bad request",
			statusCode:     http.StatusBadRequest,
			body:           "invalid format",
			expectedCode:   "BAD_REQUEST",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized",
			statusCode:     http.StatusUnauthorized,
			body:           "missing token",
			expectedCode:   "UNAUTHORIZED",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "forbidden",
			statusCode:     http.StatusForbidden,
			body:           "access denied",
			expectedCode:   "FORBIDDEN",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "not found",
			statusCode:     http.StatusNotFound,
			body:           "profile not found",
			expectedCode:   "NOT_FOUND",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "rate limited",
			statusCode:     http.StatusTooManyRequests,
			body:           "too many requests",
			expectedCode:   "RATE_LIMITED",
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "server error",
			statusCode:     http.StatusInternalServerError,
			body:           "server error",
			expectedCode:   "HYTALE_SERVER_ERROR",
			expectedStatus: http.StatusBadGateway,
		},
		{
			name:           "service unavailable",
			statusCode:     http.StatusServiceUnavailable,
			body:           "unavailable",
			expectedCode:   "SERVICE_UNAVAILABLE",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandleHTTPError(tt.statusCode, tt.body, "https://api.hytale.com/test")

			if err.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, err.Code)
			}
			if err.StatusCode != tt.expectedStatus {
				t.Errorf("expected status code %d, got %d", tt.expectedStatus, err.StatusCode)
			}
		})
	}
}

func TestHandleHTTPErrorSessionLimit(t *testing.T) {
	body := "session_limit_exceeded: 100 concurrent sessions"
	err := HandleHTTPError(http.StatusForbidden, body, "https://api.hytale.com/test")

	if err.Code != "SESSION_LIMIT_EXCEEDED" {
		t.Errorf("expected code SESSION_LIMIT_EXCEEDED, got %s", err.Code)
	}
	if !err.IsSessionLimitError {
		t.Errorf("expected IsSessionLimitError to be true")
	}
}
