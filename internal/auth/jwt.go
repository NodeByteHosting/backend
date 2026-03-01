package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// Claims represents JWT claims
type Claims struct {
	UserID             string   `json:"id"`
	Email              string   `json:"email"`
	Username           string   `json:"username"`
	FirstName          *string  `json:"firstName,omitempty"`
	LastName           *string  `json:"lastName,omitempty"`
	Roles              []string `json:"roles"`
	IsPterodactylAdmin bool     `json:"isPterodactylAdmin"`
	IsVirtfusionAdmin  bool     `json:"isVirtfusionAdmin"`
	IsSystemAdmin      bool     `json:"isSystemAdmin"`
	PterodactylID      *int     `json:"pterodactylId,omitempty"`
	EmailVerified      *string  `json:"emailVerified,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // seconds until access token expires
	TokenType    string `json:"tokenType"`
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  30 * 24 * time.Hour, // 30 days (matching NextAuth)
		refreshTokenTTL: 90 * 24 * time.Hour, // 90 days
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (s *JWTService) GenerateTokenPair(claims *Claims) (*TokenPair, error) {
	// Set expiration for access token
	now := time.Now()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(s.accessTokenTTL))
	claims.NotBefore = jwt.NewNumericDate(now)

	// Generate access token (JWT)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	// Generate refresh token (random string)
	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken validates and parses an access token
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// generateRefreshToken generates a secure random refresh token
func (s *JWTService) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetAccessTokenTTL returns the access token TTL
func (s *JWTService) GetAccessTokenTTL() time.Duration {
	return s.accessTokenTTL
}

// GetRefreshTokenTTL returns the refresh token TTL
func (s *JWTService) GetRefreshTokenTTL() time.Duration {
	return s.refreshTokenTTL
}
