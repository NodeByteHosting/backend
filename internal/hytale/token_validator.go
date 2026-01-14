package hytale

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// TokenClaims represents standard JWT claims
type TokenClaims struct {
	Sub   string                 `json:"sub"` // Subject (typically user/player ID)
	Iss   string                 `json:"iss"` // Issuer
	Aud   []string               `json:"aud"` // Audience
	Exp   int64                  `json:"exp"` // Expiration time
	Iat   int64                  `json:"iat"` // Issued at
	Jti   string                 `json:"jti"` // JWT ID (unique identifier)
	Extra map[string]interface{} `json:"-"`   // Additional claims
}

// SessionTokenClaims represents claims specific to session tokens
type SessionTokenClaims struct {
	TokenClaims
	// Add Hytale-specific claims here if needed
}

// IdentityTokenClaims represents claims specific to identity tokens
type IdentityTokenClaims struct {
	TokenClaims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	// Add other identity-specific claims as needed
}

// TokenValidator validates JWT tokens using JWKS keys
type TokenValidator struct {
	jwksCache *JWKSCache
}

// NewTokenValidator creates a new token validator
func NewTokenValidator(jwksCache *JWKSCache) *TokenValidator {
	return &TokenValidator{
		jwksCache: jwksCache,
	}
}

// ValidateSessionToken validates a session token JWT
func (tv *TokenValidator) ValidateSessionToken(ctx context.Context, tokenString string, expectedAudience string) (*SessionTokenClaims, error) {
	claims := &TokenClaims{}

	if err := tv.validateToken(ctx, tokenString, claims); err != nil {
		log.Warn().
			Err(err).
			Str("token_type", "session").
			Msg("Session token validation failed")
		return nil, err
	}

	// Verify audience if provided
	if expectedAudience != "" {
		if !contains(claims.Aud, expectedAudience) {
			log.Warn().
				Str("expected", expectedAudience).
				Strs("actual", claims.Aud).
				Msg("Session token audience mismatch")
			return nil, fmt.Errorf("invalid audience: expected %s", expectedAudience)
		}
	}

	return &SessionTokenClaims{TokenClaims: *claims}, nil
}

// ValidateIdentityToken validates an identity token JWT
func (tv *TokenValidator) ValidateIdentityToken(ctx context.Context, tokenString string, expectedAudience string) (*IdentityTokenClaims, error) {
	claims := &TokenClaims{}

	if err := tv.validateToken(ctx, tokenString, claims); err != nil {
		log.Warn().
			Err(err).
			Str("token_type", "identity").
			Msg("Identity token validation failed")
		return nil, err
	}

	// Verify audience if provided
	if expectedAudience != "" {
		if !contains(claims.Aud, expectedAudience) {
			log.Warn().
				Str("expected", expectedAudience).
				Strs("actual", claims.Aud).
				Msg("Identity token audience mismatch")
			return nil, fmt.Errorf("invalid audience: expected %s", expectedAudience)
		}
	}

	identityClaims := &IdentityTokenClaims{
		TokenClaims: *claims,
	}

	// Extract email-related claims if present
	if email, ok := claims.Extra["email"].(string); ok {
		identityClaims.Email = email
	}
	if emailVerified, ok := claims.Extra["email_verified"].(bool); ok {
		identityClaims.EmailVerified = emailVerified
	}

	return identityClaims, nil
}

// validateToken validates a JWT token's signature and expiry
func (tv *TokenValidator) validateToken(ctx context.Context, tokenString string, claims *TokenClaims) error {
	// Parse JWT header and payload
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	header, payload, signature := parts[0], parts[1], parts[2]

	// Decode header
	headerBytes, err := decodeBase64URL(header)
	if err != nil {
		return fmt.Errorf("failed to decode JWT header: %w", err)
	}

	var headerData map[string]interface{}
	if err := json.Unmarshal(headerBytes, &headerData); err != nil {
		return fmt.Errorf("failed to parse JWT header: %w", err)
	}

	// Get algorithm and key ID from header
	alg, _ := headerData["alg"].(string)
	kid, _ := headerData["kid"].(string)

	if alg != "EdDSA" {
		return fmt.Errorf("unsupported algorithm: %s (expected EdDSA)", alg)
	}

	if kid == "" {
		return fmt.Errorf("missing 'kid' in JWT header")
	}

	// Get public key from JWKS cache
	publicKey, err := tv.jwksCache.GetKey(ctx, kid)
	if err != nil {
		log.Warn().
			Str("kid", kid).
			Err(err).
			Msg("Failed to get public key for token validation")
		return fmt.Errorf("failed to get public key: %w", err)
	}

	// Verify signature
	signedMessage := header + "." + payload
	signatureBytes, err := decodeBase64URL(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if !ed25519.Verify(publicKey, []byte(signedMessage), signatureBytes) {
		log.Warn().
			Str("kid", kid).
			Msg("JWT signature verification failed")
		return fmt.Errorf("invalid JWT signature")
	}

	// Decode and parse claims
	payloadBytes, err := decodeBase64URL(payload)
	if err != nil {
		return fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Unmarshal into a generic map first to preserve extra claims
	var claimsMap map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claimsMap); err != nil {
		return fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	// Parse standard claims
	if err := json.Unmarshal(payloadBytes, claims); err != nil {
		return fmt.Errorf("failed to unmarshal standard claims: %w", err)
	}

	// Store extra claims
	claims.Extra = claimsMap

	// Verify expiry
	if claims.Exp > 0 {
		if time.Now().Unix() > claims.Exp {
			log.Warn().
				Time("expiry", time.Unix(claims.Exp, 0)).
				Msg("Token has expired")
			return fmt.Errorf("token has expired")
		}
	} else {
		return fmt.Errorf("missing 'exp' claim in token")
	}

	// Verify issued at time (shouldn't be in the future)
	if claims.Iat > 0 {
		if time.Now().Unix() < claims.Iat-60 { // Allow 60 second clock skew
			log.Warn().
				Time("issued_at", time.Unix(claims.Iat, 0)).
				Msg("Token 'iat' claim is in the future")
			return fmt.Errorf("token 'iat' claim is in the future")
		}
	}

	// Verify subject
	if claims.Sub == "" {
		return fmt.Errorf("missing 'sub' claim in token")
	}

	log.Debug().
		Str("kid", kid).
		Str("sub", claims.Sub).
		Msg("Token validation successful")

	return nil
}

// decodeBase64URL decodes a base64url-encoded string
func decodeBase64URL(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	return base64Decode(s)
}

// base64Decode decodes a base64 string (using standard or URL-safe alphabet)
func base64Decode(s string) ([]byte, error) {
	// Try URL-safe decoding first
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		return data, nil
	}

	// Fall back to standard base64
	return base64.StdEncoding.DecodeString(s)
}

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
