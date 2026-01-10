package database

import (
	"context"
)

// GetConfig retrieves a configuration value
func (db *DB) GetConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := db.Pool.QueryRow(ctx, `SELECT value FROM config WHERE key = $1`, key).Scan(&value)
	if err != nil {
		return "", nil // Return empty string on not found
	}
	return value, nil
}

// SetConfig sets a configuration value
func (db *DB) SetConfig(ctx context.Context, key, value string) error {
	// Use PostgreSQL's gen_random_uuid() for ID generation and NOW() for updatedAt
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO config (id, key, value, "updatedAt") 
		VALUES (gen_random_uuid()::text, $1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, "updatedAt" = NOW()
	`, key, value)
	return err
}

// GetAllConfigs retrieves all configuration as a map
func (db *DB) GetAllConfigs(ctx context.Context) (map[string]string, error) {
	rows, err := db.Pool.Query(ctx, `SELECT key, value FROM config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		configs[key] = value
	}

	return configs, nil
}

// HealthCheck performs a simple database health check
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
