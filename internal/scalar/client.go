package scalar

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Client is a minimal Scalar HTTP client. Replace with official SDK usage
// if you add the SDK dependency.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Scalar client
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = "https://api.scalar.com"
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetRaw performs a GET request against the provided path on the Scalar API
// and returns the raw response body as string.
func (c *Client) GetRaw(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s", c.baseURL, path), nil)
	if err != nil {
		return "", err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("scalar: unexpected status %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
