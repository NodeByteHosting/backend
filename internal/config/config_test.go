package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clean environment first
	os.Clearenv()

	tests := []struct {
		name      string
		env       map[string]string
		expectErr bool
		checkFn   func(*Config) bool
	}{
		{
			name: "missing database URL",
			env: map[string]string{
				"REDIS_URL": "redis://localhost",
			},
			expectErr: true,
		},
		{
			name: "valid minimal config",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost/db",
				"REDIS_URL":    "redis://localhost",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.DatabaseURL == "postgres://user:pass@localhost/db" &&
					cfg.Port == "8080" &&
					cfg.Env == "development"
			},
		},
		{
			name: "custom port and environment",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost/db",
				"BACKEND_PORT": "3000",
				"ENV":          "production",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.Port == "3000" && cfg.Env == "production"
			},
		},
		{
			name: "hytale staging enabled",
			env: map[string]string{
				"DATABASE_URL":       "postgres://user:pass@localhost/db",
				"HYTALE_USE_STAGING": "true",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.HytaleUseStaging == true
			},
		},
		{
			name: "hytale staging disabled",
			env: map[string]string{
				"DATABASE_URL":       "postgres://user:pass@localhost/db",
				"HYTALE_USE_STAGING": "false",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.HytaleUseStaging == false
			},
		},
		{
			name: "sync batch size parsing",
			env: map[string]string{
				"DATABASE_URL":    "postgres://user:pass@localhost/db",
				"SYNC_BATCH_SIZE": "50",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.SyncBatchSize == 50
			},
		},
		{
			name: "invalid sync batch size defaults",
			env: map[string]string{
				"DATABASE_URL":    "postgres://user:pass@localhost/db",
				"SYNC_BATCH_SIZE": "invalid",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.SyncBatchSize == 100 // default
			},
		},
		{
			name: "CORS origins configured",
			env: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost/db",
				"CORS_ORIGINS": "https://example.com,https://app.example.com",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.CORSOrigins == "https://example.com,https://app.example.com"
			},
		},
		{
			name: "email configuration",
			env: map[string]string{
				"DATABASE_URL":   "postgres://user:pass@localhost/db",
				"EMAIL_FROM":     "support@nodebyte.host",
				"RESEND_API_KEY": "test-key",
			},
			expectErr: false,
			checkFn: func(cfg *Config) bool {
				return cfg.EmailFrom == "support@nodebyte.host" &&
					cfg.ResendAPIKey == "test-key"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectErr && cfg != nil && tt.checkFn != nil {
				if !tt.checkFn(cfg) {
					t.Errorf("config check failed for %s", tt.name)
				}
			}
		})
	}
}
