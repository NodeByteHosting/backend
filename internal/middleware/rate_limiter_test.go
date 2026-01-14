package middleware

import (
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerWindow: 10,
		Window:            time.Minute,
		Identifier:        "ip",
	}

	limiter := NewRateLimiter(config)
	if limiter == nil {
		t.Fatal("expected limiter to be created")
	}
	if limiter.config.RequestsPerWindow != 10 {
		t.Errorf("expected 10 requests per window, got %d", limiter.config.RequestsPerWindow)
	}
	if limiter.config.Window != time.Minute {
		t.Errorf("expected window of 1 minute, got %v", limiter.config.Window)
	}

	// Cleanup
	limiter.Stop()
}

func TestRateLimiterAllow(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerWindow: 5,
		Window:            100 * time.Millisecond,
		Identifier:        "ip",
	}
	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	identifier := "192.168.1.1"

	// Test allowing requests within limit
	for i := 0; i < 5; i++ {
		allowed, remaining := limiter.Allow(identifier)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
		if remaining != 5-i-1 {
			t.Errorf("expected %d remaining, got %d", 5-i-1, remaining)
		}
	}

	// 6th request should be rejected
	allowed, _ := limiter.Allow(identifier)
	if allowed {
		t.Errorf("6th request should be rejected")
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should allow again after reset
	allowed, remaining := limiter.Allow(identifier)
	if !allowed {
		t.Errorf("request should be allowed after window reset")
	}
	if remaining != 4 {
		t.Errorf("expected 4 remaining after reset, got %d", remaining)
	}
}

func TestRateLimiterDifferentIdentifiers(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerWindow: 3,
		Window:            time.Second,
		Identifier:        "ip",
	}
	limiter := NewRateLimiter(config)
	defer limiter.Stop()

	// First identifier should have separate limit
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.Allow("192.168.1.1")
		if !allowed {
			t.Errorf("request %d for 192.168.1.1 should be allowed", i+1)
		}
	}

	// Second identifier should also have full limit
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.Allow("192.168.1.2")
		if !allowed {
			t.Errorf("request %d for 192.168.1.2 should be allowed", i+1)
		}
	}

	// First identifier should be exhausted
	allowed, _ := limiter.Allow("192.168.1.1")
	if allowed {
		t.Errorf("fourth request for 192.168.1.1 should be rejected")
	}

	// Second identifier should also be exhausted
	allowed, _ = limiter.Allow("192.168.1.2")
	if allowed {
		t.Errorf("fourth request for 192.168.1.2 should be rejected")
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerWindow: 1,
		Window:            10 * time.Millisecond,
		Identifier:        "ip",
	}
	limiter := NewRateLimiter(config)

	// Make a request to create a bucket
	limiter.Allow("192.168.1.1")

	if len(limiter.buckets) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(limiter.buckets))
	}

	// Stop should clean up
	limiter.Stop()

	if len(limiter.buckets) != 0 {
		t.Errorf("expected 0 buckets after stop, got %d", len(limiter.buckets))
	}
}

// Test predefined rate limit configurations
func TestPredefinedRateLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		minReqs int
	}{
		{
			name:    "device code limit",
			config:  DeviceCodeRateLimit,
			minReqs: 5,
		},
		{
			name:    "token poll limit",
			config:  TokenPollRateLimit,
			minReqs: 10,
		},
		{
			name:    "token refresh limit",
			config:  TokenRefreshRateLimit,
			minReqs: 6,
		},
		{
			name:    "game session limit",
			config:  GameSessionRateLimit,
			minReqs: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.RequestsPerWindow != tt.minReqs {
				t.Errorf("expected %d requests, got %d", tt.minReqs, tt.config.RequestsPerWindow)
			}
			if tt.config.Window == 0 {
				t.Errorf("expected non-zero window duration")
			}
		})
	}
}
