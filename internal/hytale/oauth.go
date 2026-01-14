package hytale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// OAuthClientConfig holds Hytale OAuth configuration
type OAuthClientConfig struct {
	ClientID   string
	UseStaging bool // If true, use arcanitegames.ca instead of hytale.com
}

// OAuthClient handles communication with Hytale OAuth endpoints
type OAuthClient struct {
	config *OAuthClientConfig
	client *http.Client
}

// NewOAuthClient creates a new Hytale OAuth client
func NewOAuthClient(config *OAuthClientConfig) *OAuthClient {
	return &OAuthClient{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DeviceCodeResponse represents the response from /oauth2/device/auth
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// TokenResponse represents the response from /oauth2/token
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	Scope            string `json:"scope"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// RequestDeviceCode requests a device code from Hytale OAuth
func (c *OAuthClient) RequestDeviceCode(ctx context.Context) (*DeviceCodeResponse, error) {
	endpoint := c.getOAuthEndpoint("/oauth2/device/auth")

	data := url.Values{}
	data.Set("client_id", c.config.ClientID)
	data.Set("scope", "openid offline auth:server")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hytale returned %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}

	return &deviceResp, nil
}

// PollToken polls the token endpoint with a device code
func (c *OAuthClient) PollToken(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	endpoint := c.getOAuthEndpoint("/oauth2/token")

	data := url.Values{}
	data.Set("client_id", c.config.ClientID)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", deviceCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to poll token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken exchanges a refresh token for a new access token
func (c *OAuthClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	endpoint := c.getOAuthEndpoint("/oauth2/token")

	data := url.Values{}
	data.Set("client_id", c.config.ClientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hytale returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// getOAuthEndpoint constructs the full OAuth endpoint URL
func (c *OAuthClient) getOAuthEndpoint(path string) string {
	host := "hytale.com"
	if c.config.UseStaging {
		host = "arcanitegames.ca"
	}
	return fmt.Sprintf("https://oauth.accounts.%s%s", host, path)
}

// GetProfilesResponse represents the response from /my-account/get-profiles
type GetProfilesResponse struct {
	Owner    string        `json:"owner"`
	Profiles []GameProfile `json:"profiles"`
}

// GameProfile represents a game profile
type GameProfile struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
}

// GetProfiles fetches available game profiles
func (c *OAuthClient) GetProfiles(ctx context.Context, accessToken string) (*GetProfilesResponse, error) {
	endpoint := c.getAccountDataEndpoint("/my-account/get-profiles")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hytale returned %d: %s", resp.StatusCode, string(body))
	}

	var profileResp GetProfilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&profileResp); err != nil {
		return nil, fmt.Errorf("failed to decode profiles response: %w", err)
	}

	return &profileResp, nil
}

// GameSessionResponse represents the response from /game-session/new
type GameSessionResponse struct {
	SessionToken  string `json:"sessionToken"`
	IdentityToken string `json:"identityToken"`
	ExpiresAt     string `json:"expiresAt"`
}

// CreateGameSession creates a new game session
func (c *OAuthClient) CreateGameSession(ctx context.Context, accessToken string, profileUUID string) (*GameSessionResponse, error) {
	endpoint := c.getSessionEndpoint("/game-session/new")

	body := map[string]string{"uuid": profileUUID}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create game session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hytale returned %d: %s", resp.StatusCode, string(body))
	}

	var sessionResp GameSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode game session response: %w", err)
	}

	return &sessionResp, nil
}

// RefreshGameSession refreshes an existing game session
func (c *OAuthClient) RefreshGameSession(ctx context.Context, sessionToken string) error {
	endpoint := c.getSessionEndpoint("/game-session/refresh")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sessionToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh game session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hytale returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TerminateGameSession terminates a game session
func (c *OAuthClient) TerminateGameSession(ctx context.Context, sessionToken string) error {
	endpoint := c.getSessionEndpoint("/game-session")

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sessionToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to terminate game session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hytale returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getAccountDataEndpoint constructs the full account data endpoint URL
func (c *OAuthClient) getAccountDataEndpoint(path string) string {
	host := "hytale.com"
	if c.config.UseStaging {
		host = "arcanitegames.ca"
	}
	return fmt.Sprintf("https://account-data.%s%s", host, path)
}

// getSessionEndpoint constructs the full session endpoint URL
func (c *OAuthClient) getSessionEndpoint(path string) string {
	host := "hytale.com"
	if c.config.UseStaging {
		host = "arcanitegames.ca"
	}
	return fmt.Sprintf("https://sessions.%s%s", host, path)
}
