package hytale

import (
	"testing"
)

func TestNewJWKSCache(t *testing.T) {
	tests := []struct {
		name        string
		useStaging  bool
		expectedURL string
	}{
		{
			name:        "production cache",
			useStaging:  false,
			expectedURL: "https://sessions.hytale.com/.well-known/jwks.json",
		},
		{
			name:        "staging cache",
			useStaging:  true,
			expectedURL: "https://sessions.arcanitegames.ca/.well-known/jwks.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewJWKSCache(tt.useStaging)
			if cache == nil {
				t.Fatal("expected cache to be created")
			}
			if cache.jwksURL != tt.expectedURL {
				t.Errorf("expected URL %s, got %s", tt.expectedURL, cache.jwksURL)
			}
		})
	}
}

func TestJWKSCacheInitialization(t *testing.T) {
	cache := NewJWKSCache(false)

	if cache == nil {
		t.Fatal("expected cache to be created")
	}
	if cache.keys == nil {
		t.Errorf("expected keys map to be initialized")
	}
	if cache.rawKeys == nil {
		t.Errorf("expected rawKeys map to be initialized")
	}
	if cache.client == nil {
		t.Errorf("expected HTTP client to be initialized")
	}
	if cache.refreshInterval == 0 {
		t.Errorf("expected refresh interval to be set")
	}
}

func TestJWKStructure(t *testing.T) {
	jwk := JWK{
		Kty: "OKP",
		Crv: "Ed25519",
		X:   "test-public-key",
		Kid: "key-123",
		Use: "sig",
		Alg: "EdDSA",
	}

	if jwk.Kty != "OKP" {
		t.Errorf("expected kty OKP")
	}
	if jwk.Crv != "Ed25519" {
		t.Errorf("expected curve Ed25519")
	}
	if jwk.Kid != "key-123" {
		t.Errorf("expected kid key-123")
	}
	if jwk.Use != "sig" {
		t.Errorf("expected use sig")
	}
	if jwk.X != "test-public-key" {
		t.Errorf("expected X to be test-public-key")
	}
}

func TestJWKSKeySetStructure(t *testing.T) {
	keySet := JWKSKeySet{
		Keys: []JWK{
			{
				Kty: "OKP",
				Crv: "Ed25519",
				Kid: "key-1",
				Use: "sig",
				Alg: "EdDSA",
			},
			{
				Kty: "OKP",
				Crv: "Ed25519",
				Kid: "key-2",
				Use: "sig",
				Alg: "EdDSA",
			},
		},
	}

	if len(keySet.Keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keySet.Keys))
	}

	if keySet.Keys[0].Kid != "key-1" {
		t.Errorf("expected first key to be key-1")
	}
	if keySet.Keys[1].Kid != "key-2" {
		t.Errorf("expected second key to be key-2")
	}
}

func TestJWKWithOptionalFields(t *testing.T) {
	jwk := JWK{
		Kty:    "OKP",
		Crv:    "Ed25519",
		X:      "public-key",
		Kid:    "key-id",
		Use:    "sig",
		Alg:    "EdDSA",
		Exp:    1704067200,
		KeyOps: []string{"verify"},
	}

	if jwk.Exp != 1704067200 {
		t.Errorf("expected expiration timestamp 1704067200, got %d", jwk.Exp)
	}
	if len(jwk.KeyOps) != 1 || jwk.KeyOps[0] != "verify" {
		t.Errorf("expected KeyOps to contain 'verify'")
	}
}
