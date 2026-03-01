package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hibiken/asynq"
)

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// ParseRedisURL parses a Redis connection string and returns configuration.
// Supports formats: redis://[user:pass@]host:port/db or host:port
func ParseRedisURL(redisURL string) (*RedisConfig, error) {
	// Handle simple host:port format
	if !strings.Contains(redisURL, "://") {
		parts := strings.Split(redisURL, ":")
		if len(parts) == 2 {
			return &RedisConfig{
				Addr: redisURL,
				DB:   0,
			}, nil
		}
	}

	// Parse full redis:// URL
	u, err := url.Parse(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}

	// Get host and port
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "6379"
	}
	addr := host + ":" + port

	// Get credentials
	var password string
	if u.User != nil {
		password, _ = u.User.Password()
	}

	// Get database number
	db := 0
	if u.Path != "" {
		path := strings.TrimPrefix(u.Path, "/")
		if path != "" {
			if dbNum, err := strconv.Atoi(path); err == nil {
				db = dbNum
			}
		}
	}

	return &RedisConfig{
		Addr:     addr,
		Password: password,
		DB:       db,
	}, nil
}

// ToAsynqOpt converts the Redis config to an Asynq client option.
func (c *RedisConfig) ToAsynqOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	}
}
