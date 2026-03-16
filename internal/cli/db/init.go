package db

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// InitCmd handles database initialization.
type InitCmd struct {
	DatabaseURL string
	SchemasDir  string
}

// NewInitCmd creates a new init command with parsed flags.
func NewInitCmd(args []string) (*InitCmd, error) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.Usage = func() {}
	databaseURL := fs.String("database", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	schemasDir := fs.String("schemas", "", "Path to schemas directory (optional)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *databaseURL == "" {
		return nil, fmt.Errorf("no database URL provided (use -database flag or DATABASE_URL env var)")
	}

	return &InitCmd{
		DatabaseURL: *databaseURL,
		SchemasDir:  *schemasDir,
	}, nil
}

// Run executes the init command.
func (c *InitCmd) Run(ctx context.Context) error {
	client, err := New(ctx, c.DatabaseURL, c.SchemasDir)
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	fmt.Println("============================================================================")
	fmt.Println("NodeByte Database Initialization")
	fmt.Println("============================================================================")
	fmt.Println()
	fmt.Printf("📦 Initializing database with %d schemas...\n", len(SchemaList))
	fmt.Println()

	if err := client.MigrateAll(ctx); err != nil {
		fmt.Printf("❌ Migration failed: %v\n", err)
		return err
	}

	fmt.Println()
	fmt.Println("============================================================================")
	fmt.Println("✅ Database initialization complete!")
	fmt.Println("============================================================================")
	fmt.Println()

	return nil
}
