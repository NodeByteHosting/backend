package hytale

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// JWKSKeySet represents a JWKS key set
type JWKSKeySet struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty    string   `json:"kty"`     // Key Type (e.g., "OKP")
	Crv    string   `json:"crv"`     // Curve (e.g., "Ed25519")
	X      string   `json:"x"`       // Public key material (base64url encoded)
	Use    string   `json:"use"`     // Public Key Use
	Kid    string   `json:"kid"`     // Key ID
	Alg    string   `json:"alg"`     // Algorithm
	Exp    int64    `json:"exp"`     // Expiration time (Unix timestamp)
	KeyOps []string `json:"key_ops"` // Allowed operations
}

// JWKSCache manages JWKS key caching with periodic refresh
type JWKSCache struct {
	mu              sync.RWMutex
	keys            map[string]ed25519.PublicKey
	rawKeys         map[string]JWK
	jwksURL         string
	lastRefresh     time.Time
	refreshInterval time.Duration
	client          *http.Client
}

// NewJWKSCache creates a new JWKS cache with the given configuration
func NewJWKSCache(useStaging bool) *JWKSCache {
	endpoint := "https://sessions.hytale.com"
	if useStaging {
		endpoint = "https://sessions.arcanitegames.ca"
	}

	return &JWKSCache{
		keys:            make(map[string]ed25519.PublicKey),
		rawKeys:         make(map[string]JWK),
		jwksURL:         endpoint + "/.well-known/jwks.json",
		refreshInterval: 1 * time.Hour, // Refresh every hour
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Refresh fetches the latest JWKS from the endpoint
func (c *JWKSCache) Refresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.jwksURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", c.jwksURL).Msg("Failed to create JWKS request")
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", c.jwksURL).Msg("Failed to fetch JWKS")
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("body", string(body)).
			Str("url", c.jwksURL).
			Msg("JWKS endpoint returned non-200 status")
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var keySet JWKSKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		log.Error().Err(err).Msg("Failed to decode JWKS response")
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Clear old keys
	c.keys = make(map[string]ed25519.PublicKey)
	c.rawKeys = make(map[string]JWK)

	// Process each key in the set
	for _, jwk := range keySet.Keys {
		if jwk.Kty == "OKP" && jwk.Crv == "Ed25519" {
			// Decode the base64url-encoded public key
			keyBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
			if err != nil {
				log.Warn().
					Err(err).
					Str("kid", jwk.Kid).
					Msg("Failed to decode JWKS key material")
				continue
			}

			if len(keyBytes) != 32 {
				log.Warn().
					Str("kid", jwk.Kid).
					Int("length", len(keyBytes)).
					Msg("Invalid Ed25519 public key length")
				continue
			}

			publicKey := ed25519.PublicKey(keyBytes)
			keyID := jwk.Kid
			if keyID == "" {
				keyID = jwk.X // Use key material as fallback ID
			}

			c.keys[keyID] = publicKey
			c.rawKeys[keyID] = jwk
			log.Debug().
				Str("kid", keyID).
				Str("alg", jwk.Alg).
				Msg("Loaded JWKS key")
		}
	}

	c.lastRefresh = time.Now()
	log.Info().
		Int("keys_loaded", len(c.keys)).
		Time("last_refresh", c.lastRefresh).
		Msg("JWKS cache refreshed successfully")

	return nil
}

// GetKey retrieves a public key by key ID
func (c *JWKSCache) GetKey(ctx context.Context, keyID string) (ed25519.PublicKey, error) {
	c.mu.RLock()

	// Check if we need to refresh
	needsRefresh := c.lastRefresh.IsZero() || time.Since(c.lastRefresh) > c.refreshInterval

	if !needsRefresh {
		if key, exists := c.keys[keyID]; exists {
			defer c.mu.RUnlock()
			return key, nil
		}
	}

	c.mu.RUnlock()

	// If not found or needs refresh, fetch new keys
	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if key, exists := c.keys[keyID]; exists {
		return key, nil
	}

	return nil, fmt.Errorf("key not found: %s", keyID)
}

// GetAllKeys returns all cached public keys
func (c *JWKSCache) GetAllKeys() map[string]ed25519.PublicKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	keys := make(map[string]ed25519.PublicKey)
	for k, v := range c.keys {
		keys[k] = v
	}
	return keys
}

// IsExpired checks if the cache is expired
func (c *JWKSCache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastRefresh.IsZero() {
		return true
	}
	return time.Since(c.lastRefresh) > c.refreshInterval
}

// LastRefreshTime returns the timestamp of the last refresh
func (c *JWKSCache) LastRefreshTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRefresh
}
