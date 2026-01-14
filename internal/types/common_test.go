package types

import (
	"testing"
)

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		error   string
	}{
		{
			name:    "error response",
			success: false,
			error:   "Invalid credentials",
		},
		{
			name:    "success response",
			success: true,
			error:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ErrorResponse{
				Success: tt.success,
				Error:   tt.error,
			}

			if resp.Success != tt.success {
				t.Errorf("expected success=%v, got %v", tt.success, resp.Success)
			}
			if resp.Error != tt.error {
				t.Errorf("expected error=%q, got %q", tt.error, resp.Error)
			}
		})
	}
}

func TestSuccessResponse(t *testing.T) {
	resp := SuccessResponse{
		Success: true,
		Message: "Operation completed successfully",
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
	if resp.Message != "Operation completed successfully" {
		t.Errorf("expected message 'Operation completed successfully', got %s", resp.Message)
	}
}

func TestSuccessResponseWithoutMessage(t *testing.T) {
	resp := SuccessResponse{
		Success: true,
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
	if resp.Message != "" {
		t.Errorf("expected empty message, got %s", resp.Message)
	}
}

func TestErrorResponseZeroValues(t *testing.T) {
	resp := ErrorResponse{}
	if resp.Success {
		t.Errorf("zero value success should be false")
	}
	if resp.Error != "" {
		t.Errorf("zero value error should be empty string")
	}
}

func TestSuccessResponseZeroValues(t *testing.T) {
	resp := SuccessResponse{}
	if resp.Success {
		t.Errorf("zero value success should be false")
	}
	if resp.Message != "" {
		t.Errorf("zero value message should be empty string")
	}
}
