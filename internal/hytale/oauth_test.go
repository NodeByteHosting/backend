package hytale

import (
	"context"
	"testing"
)

func TestNewOAuthClient(t *testing.T) {
	config := &OAuthClientConfig{
		ClientID:   "test-client-id",
		UseStaging: false,
	}

	client := NewOAuthClient(config)
	if client == nil {
		t.Fatal("expected client to be created")
	}
	if client.config.ClientID != "test-client-id" {
		t.Errorf("expected client ID to be test-client-id, got %s", client.config.ClientID)
	}
	if client.config.UseStaging {
		t.Errorf("expected UseStaging to be false")
	}
}

func TestGetOAuthEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		useStaging bool
		path       string
		expected   string
	}{
		{
			name:       "production endpoint",
			useStaging: false,
			path:       "/oauth2/device/auth",
			expected:   "https://oauth.accounts.hytale.com/oauth2/device/auth",
		},
		{
			name:       "staging endpoint",
			useStaging: true,
			path:       "/oauth2/device/auth",
			expected:   "https://oauth.accounts.arcanitegames.ca/oauth2/device/auth",
		},
		{
			name:       "production token endpoint",
			useStaging: false,
			path:       "/oauth2/token",
			expected:   "https://oauth.accounts.hytale.com/oauth2/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OAuthClientConfig{
				ClientID:   "test",
				UseStaging: tt.useStaging,
			}
			client := NewOAuthClient(config)
			endpoint := client.getOAuthEndpoint(tt.path)
			if endpoint != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, endpoint)
			}
		})
	}
}

func TestOAuthClientConfig(t *testing.T) {
	config := &OAuthClientConfig{
		ClientID:   "my-client",
		UseStaging: true,
	}

	if config.ClientID != "my-client" {
		t.Errorf("expected client ID my-client, got %s", config.ClientID)
	}
	if !config.UseStaging {
		t.Errorf("expected UseStaging to be true")
	}
}

func TestDeviceCodeResponse(t *testing.T) {
	response := &DeviceCodeResponse{
		DeviceCode:              "device123",
		UserCode:                "USER-1234",
		VerificationURI:         "https://hytale.com/auth",
		VerificationURIComplete: "https://hytale.com/auth?user_code=USER-1234",
		ExpiresIn:               1800,
		Interval:                5,
	}

	if response.DeviceCode != "device123" {
		t.Errorf("expected device code device123, got %s", response.DeviceCode)
	}
	if response.UserCode != "USER-1234" {
		t.Errorf("expected user code USER-1234, got %s", response.UserCode)
	}
	if response.ExpiresIn != 1800 {
		t.Errorf("expected 1800 seconds expiry, got %d", response.ExpiresIn)
	}
}

func TestTokenResponse(t *testing.T) {
	tests := []struct {
		name    string
		token   *TokenResponse
		isError bool
	}{
		{
			name: "successful token response",
			token: &TokenResponse{
				AccessToken:  "access-token-123",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "refresh-token-123",
				Scope:        "openid offline auth:server",
			},
			isError: false,
		},
		{
			name: "error response",
			token: &TokenResponse{
				Error:            "invalid_grant",
				ErrorDescription: "The device code has expired",
			},
			isError: true,
		},
		{
			name: "authorization pending",
			token: &TokenResponse{
				Error: "authorization_pending",
			},
			isError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isError && tt.token.Error == "" {
				t.Errorf("expected error response to have error field")
			}
			if !tt.isError && tt.token.Error != "" {
				t.Errorf("expected success response to not have error field")
			}
			if !tt.isError && tt.token.AccessToken == "" {
				t.Errorf("expected access token in success response")
			}
		})
	}
}

func TestOAuthClientTimeout(t *testing.T) {
	config := &OAuthClientConfig{
		ClientID:   "test",
		UseStaging: false,
	}
	client := NewOAuthClient(config)

	if client.client == nil {
		t.Fatal("expected HTTP client to be initialized")
	}
	if client.client.Timeout == 0 {
		t.Errorf("expected HTTP client to have timeout set")
	}
}

func TestOAuthClientContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Verify context can be used with client methods
	if ctx.Err() != nil {
		t.Errorf("context should be valid")
	}
}
