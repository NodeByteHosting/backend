package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/types"
)

// RateLimitConfig defines rate limit configuration for an endpoint
type RateLimitConfig struct {
	RequestsPerWindow int           // Number of requests allowed
	Window            time.Duration // Time window for rate limiting
	Identifier        string        // "ip" or "account_id"
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	config  RateLimitConfig
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
	ticker  *time.Ticker
	done    chan struct{}
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	limiter := &RateLimiter{
		config:  config,
		buckets: make(map[string]*TokenBucket),
		done:    make(chan struct{}),
	}

	// Start cleanup goroutine to remove stale buckets
	limiter.ticker = time.NewTicker(5 * time.Minute)
	go limiter.cleanupStale()

	return limiter
}

// Middleware returns a Fiber middleware handler for rate limiting
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		identifier := rl.getIdentifier(c)
		allowed, remaining := rl.Allow(identifier)

		// Add rate limit headers
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerWindow))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(rl.config.Window).Unix()))

		if !allowed {
			retryAfter := int(rl.config.Window.Seconds())
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))

			log.Warn().
				Str("identifier", identifier).
				Str("endpoint", c.Path()).
				Str("method", c.Method()).
				Int("limit", rl.config.RequestsPerWindow).
				Msg("Rate limit exceeded")

			return c.Status(http.StatusTooManyRequests).JSON(types.ErrorResponse{
				Success: false,
				Error:   fmt.Sprintf("Rate limit exceeded. Maximum %d requests per %v. Retry after %d seconds.", rl.config.RequestsPerWindow, rl.config.Window, retryAfter),
			})
		}

		return c.Next()
	}
}

// Allow checks if a request from the given identifier is allowed
func (rl *RateLimiter) Allow(identifier string) (bool, int) {
	rl.mu.Lock()
	bucket, exists := rl.buckets[identifier]
	if !exists {
		bucket = rl.newTokenBucket()
		rl.buckets[identifier] = bucket
	}
	rl.mu.Unlock()

	return bucket.Allow()
}

// newTokenBucket creates a new token bucket
func (rl *RateLimiter) newTokenBucket() *TokenBucket {
	capacity := float64(rl.config.RequestsPerWindow)
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: capacity / rl.config.Window.Seconds(),
		lastRefill: time.Now(),
	}
}

// Allow checks if a token is available and consumes it if so
func (tb *TokenBucket) Allow() (bool, int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(tb.capacity, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true, int(tb.tokens)
	}

	return false, 0
}

// getIdentifier returns the identifier for rate limiting (IP or account ID)
func (rl *RateLimiter) getIdentifier(c *fiber.Ctx) string {
	if rl.config.Identifier == "account_id" {
		// Try to get account_id from request body or context
		if accountID := c.Locals("account_id"); accountID != nil {
			return fmt.Sprintf("account:%v", accountID)
		}
		// Parse from request body if available
		var body map[string]interface{}
		if err := c.BodyParser(&body); err == nil {
			if accountID, ok := body["account_id"]; ok {
				return fmt.Sprintf("account:%v", accountID)
			}
		}
	}

	// Default to IP-based rate limiting
	return c.IP()
}

// Stop stops the cleanup goroutine
func (rl *RateLimiter) Stop() {
	if rl.ticker != nil {
		rl.ticker.Stop()
	}
	close(rl.done)
}

// cleanupStale removes stale buckets that haven't been used
func (rl *RateLimiter) cleanupStale() {
	for {
		select {
		case <-rl.done:
			return
		case <-rl.ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for id, bucket := range rl.buckets {
				bucket.mu.Lock()
				if now.Sub(bucket.lastRefill) > 30*time.Minute {
					delete(rl.buckets, id)
				}
				bucket.mu.Unlock()
			}
			rl.mu.Unlock()
		}
	}
}

// min returns the minimum of two numbers
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Predefined rate limit configurations for Hytale OAuth endpoints
var (
	// DeviceCodeRateLimit: 5 requests per 15 minutes per IP
	DeviceCodeRateLimit = RateLimitConfig{
		RequestsPerWindow: 5,
		Window:            15 * time.Minute,
		Identifier:        "ip",
	}

	// TokenPollRateLimit: 10 requests per 5 minutes per account
	TokenPollRateLimit = RateLimitConfig{
		RequestsPerWindow: 10,
		Window:            5 * time.Minute,
		Identifier:        "account_id",
	}

	// TokenRefreshRateLimit: 6 requests per hour per account
	TokenRefreshRateLimit = RateLimitConfig{
		RequestsPerWindow: 6,
		Window:            1 * time.Hour,
		Identifier:        "account_id",
	}

	// GameSessionRateLimit: 20 requests per hour per account
	GameSessionRateLimit = RateLimitConfig{
		RequestsPerWindow: 20,
		Window:            1 * time.Hour,
		Identifier:        "account_id",
	}
)
