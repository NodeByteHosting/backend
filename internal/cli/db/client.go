package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
)

// Client handles database migration operations.
type Client struct {
	conn       *pgx.Conn
	schemasDir string
}

// New creates a new database client and establishes a connection.
func New(ctx context.Context, databaseURL, schemasDir string) (*Client, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// If schemasDir is empty, find it automatically
	if schemasDir == "" {
		schemasDir = findSchemasDir()
		if schemasDir == "" {
			conn.Close(ctx)
			return nil, fmt.Errorf("schemas directory not found")
		}
	}

	return &Client{
		conn:       conn,
		schemasDir: schemasDir,
	}, nil
}

// Close closes the database connection.
func (c *Client) Close(ctx context.Context) error {
	if c.conn != nil {
		return c.conn.Close(ctx)
	}
	return nil
}

// Migrate applies a single schema file to the database.
func (c *Client) Migrate(ctx context.Context, schemaFile string) error {
	filePath := filepath.Join(c.schemasDir, schemaFile)

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("schema file not found: %s", filePath)
	}

	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read schema file: %w", err)
	}

	if _, err := c.conn.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	return nil
}

// MigrateAll applies all schemas to the database.
func (c *Client) MigrateAll(ctx context.Context) error {
	for _, schema := range SchemaList {
		if err := c.Migrate(ctx, schema); err != nil {
			return fmt.Errorf("migrate %s: %w", schema, err)
		}
	}
	return nil
}

// ValidateSchema checks if a schema file exists.
func (c *Client) ValidateSchema(schema string) error {
	filePath := filepath.Join(c.schemasDir, schema)
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("schema file not found: %s", filePath)
	}
	return nil
}

// findSchemasDir attempts to locate the schemas directory.
func findSchemasDir() string {
	possiblePaths := []string{
		"./schemas",
		"./backend/schemas",
		"../schemas",
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}

	return ""
}
