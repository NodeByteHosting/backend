package db

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ResetCmd handles database reset operations.
type ResetCmd struct {
	DatabaseURL string
	SchemasDir  string
}

// NewResetCmd creates a new reset command with parsed flags.
func NewResetCmd(args []string) (*ResetCmd, error) {
	fs := flag.NewFlagSet("reset", flag.ContinueOnError)
	fs.Usage = func() {}
	databaseURL := fs.String("database", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	schemasDir := fs.String("schemas", "", "Path to schemas directory (optional)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *databaseURL == "" {
		return nil, fmt.Errorf("no database URL provided (use -database flag or DATABASE_URL env var)")
	}

	return &ResetCmd{
		DatabaseURL: *databaseURL,
		SchemasDir:  *schemasDir,
	}, nil
}

// Run executes the reset command.
func (c *ResetCmd) Run(ctx context.Context) error {
	// Parse database name from connection string
	connConfig, err := pgx.ParseConfig(c.DatabaseURL)
	if err != nil {
		return fmt.Errorf("parse database URL: %w", err)
	}

	dbName := connConfig.Database

	// Confirm before resetting
	if !c.confirmReset(dbName) {
		fmt.Println("❌ Confirmation failed. Database not reset.")
		return nil
	}

	// Connect to postgres (not the target database)
	pgConfig := *connConfig
	pgConfig.Database = "postgres"
	conn, err := pgx.ConnectConfig(ctx, &pgConfig)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer conn.Close(ctx)

	fmt.Println()
	fmt.Printf("🔄 Dropping database '%s'...\n", dbName)
	if _, err := conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName)); err != nil {
		return fmt.Errorf("drop database: %w", err)
	}

	fmt.Printf("🔄 Creating database '%s'...\n", dbName)
	if _, err := conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s;", dbName)); err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	// Connect to the new database
	newConn, err := pgx.Connect(ctx, c.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to new database: %w", err)
	}
	defer newConn.Close(ctx)

	fmt.Println()
	fmt.Printf("📦 Initializing database with %d schemas...\n", len(SchemaList))
	fmt.Println()

	client := &Client{
		conn:       newConn,
		schemasDir: findSchemasDir(),
	}

	if client.schemasDir == "" && c.SchemasDir != "" {
		client.schemasDir = c.SchemasDir
	}

	if client.schemasDir == "" {
		return fmt.Errorf("schemas directory not found")
	}

	for _, schema := range SchemaList {
		if err := client.Migrate(ctx, schema); err != nil {
			fmt.Printf("❌ %s\n   Error: %v\n", schema, err)
			return err
		}
		fmt.Printf("✅ %s\n", schema)
	}

	fmt.Println()
	fmt.Println("============================================================================")
	fmt.Println("✅ Database reset and initialization complete!")
	fmt.Println("============================================================================")
	fmt.Println()

	return nil
}

// confirmReset prompts the user to confirm database reset.
func (c *ResetCmd) confirmReset(dbName string) bool {
	fmt.Printf("⚠️  WARNING: This will DROP and recreate the database '%s'\n", dbName)
	fmt.Print("Are you SURE? Type the database name to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	return input == dbName
}
