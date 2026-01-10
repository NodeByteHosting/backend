package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/nodebyte/backend/internal/crypto"
	"github.com/nodebyte/backend/internal/database"
)

// Config holds all configuration for the backend service
type Config struct {
	// Environment
	Env  string
	Port string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Security
	APIKey string

	// CORS
	CORSOrigins string

	// Pterodactyl Panel
	PterodactylURL          string
	PterodactylAPIKey       string
	PterodactylClientAPIKey string

	// Virtfusion Panel
	VirtfusionURL    string
	VirtfusionAPIKey string

	// Cloudflare Access (for panel proxying)
	CFAccessClientID     string
	CFAccessClientSecret string

	// Email (Resend)
	ResendAPIKey string
	EmailFrom    string

	// Sync settings
	SyncBatchSize    int
	AutoSyncEnabled  bool
	AutoSyncInterval int // in seconds (loaded from database or env; env can be in minutes/seconds)

	// Scalar (feature/config platform)
	ScalarURL    string
	ScalarAPIKey string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Env:         getEnv("ENV", "development"),
		Port:        getEnv("BACKEND_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		APIKey:      os.Getenv("BACKEND_API_KEY"),
		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:3000"),

		// Panel settings
		PterodactylURL:          os.Getenv("PTERODACTYL_URL"),
		PterodactylAPIKey:       os.Getenv("PTERODACTYL_API_KEY"),
		PterodactylClientAPIKey: os.Getenv("PTERODACTYL_CLIENT_API_KEY"),
		VirtfusionURL:           os.Getenv("VIRTFUSION_URL"),
		VirtfusionAPIKey:        os.Getenv("VIRTFUSION_API_KEY"),

		// Cloudflare
		CFAccessClientID:     os.Getenv("CF_ACCESS_CLIENT_ID"),
		CFAccessClientSecret: os.Getenv("CF_ACCESS_CLIENT_SECRET"),

		// Email
		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
		EmailFrom:    getEnv("EMAIL_FROM", "NodeByte <noreply@nodebyte.host>"),

		// Sync
		SyncBatchSize:    getEnvInt("SYNC_BATCH_SIZE", 100),
		AutoSyncEnabled:  getEnvBool("AUTO_SYNC_ENABLED", false),
		AutoSyncInterval: getEnvInt("AUTO_SYNC_INTERVAL", 3600) * 60, // Env in minutes (converted to seconds)
		// Scalar
		ScalarURL:    os.Getenv("SCALAR_URL"),
		ScalarAPIKey: os.Getenv("SCALAR_API_KEY"),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// MergeFromDB loads configuration overrides from the `config` table in the
// main application database. Values stored in the DB will overwrite the
// corresponding fields on the provided Config when present.
// Sensitive fields (API keys) will be decrypted if an encryptor is provided.
func (cfg *Config) MergeFromDB(db *database.DB, encryptor *crypto.Encryptor) error {
	ctx := context.Background()
	rows, err := db.Pool.Query(ctx, `SELECT key, value FROM config`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// List of sensitive fields that may be encrypted
	sensitiveFields := map[string]bool{
		"pterodactyl_api_key":        true,
		"pterodactyl_client_api_key": true,
		"virtfusion_api_key":         true,
		"resend_api_key":             true,
		"cf_access_client_secret":    true,
		"scalar_api_key":             true,
	}

	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}

		// Decrypt sensitive fields if encryptor is available
		if isSensitive := sensitiveFields[key]; isSensitive && encryptor != nil {
			if decrypted, err := encryptor.Decrypt(value); err == nil {
				value = decrypted
			} else {
				// Decryption failed - log warning and try using raw value
				// This can happen if ENCRYPTION_KEY is not set or key changed
				fmt.Printf("WARNING: Failed to decrypt %s from database, using raw value: %v\n", key, err)
			}
		} else if isSensitive && encryptor == nil {
			fmt.Printf("WARNING: Sensitive field '%s' in database but encryptor not available. Value may be encrypted.\n", key)
		}

		switch key {
		case "pterodactyl_url":
			if value != "" {
				cfg.PterodactylURL = value
			}
		case "pterodactyl_api_key":
			if value != "" {
				cfg.PterodactylAPIKey = value
			}
		case "pterodactyl_client_api_key":
			if value != "" {
				cfg.PterodactylClientAPIKey = value
			}
		case "virtfusion_url":
			if value != "" {
				cfg.VirtfusionURL = value
			}
		case "virtfusion_api_key":
			if value != "" {
				cfg.VirtfusionAPIKey = value
			}
		case "cf_access_client_id":
			if value != "" {
				cfg.CFAccessClientID = value
			}
		case "cf_access_client_secret":
			if value != "" {
				cfg.CFAccessClientSecret = value
			}
		case "resend_api_key":
			if value != "" {
				cfg.ResendAPIKey = value
			}
		case "email_from":
			if value != "" {
				cfg.EmailFrom = value
			}
		case "sync_batch_size":
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				cfg.SyncBatchSize = n
			}
		case "auto_sync_enabled":
			cfg.AutoSyncEnabled = (value == "true" || value == "1")
		case "auto_sync_interval":
			// Database stores interval in seconds
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				cfg.AutoSyncInterval = n
			}
		case "scalar_url":
			if value != "" {
				cfg.ScalarURL = value
			}
		case "scalar_api_key":
			if value != "" {
				cfg.ScalarAPIKey = value
			}
		}
	}

	return nil
}
