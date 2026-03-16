package db

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MigrateCmd handles database schema migration.
type MigrateCmd struct {
	DatabaseURL string
	SchemaFile  string
	SchemasDir  string
	Interactive bool
}

// NewMigrateCmd creates a new migrate command with parsed flags.
func NewMigrateCmd(args []string) (*MigrateCmd, error) {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.Usage = func() {}
	databaseURL := fs.String("database", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	schemaFile := fs.String("schema", "", "Specific schema file to migrate (optional)")
	schemasDir := fs.String("schemas", "", "Path to schemas directory (optional)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *databaseURL == "" {
		return nil, fmt.Errorf("no database URL provided (use -database flag or DATABASE_URL env var)")
	}

	return &MigrateCmd{
		DatabaseURL: *databaseURL,
		SchemaFile:  *schemaFile,
		SchemasDir:  *schemasDir,
		Interactive: *schemaFile == "",
	}, nil
}

// Run executes the migrate command.
func (c *MigrateCmd) Run(ctx context.Context) error {
	client, err := New(ctx, c.DatabaseURL, c.SchemasDir)
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	fmt.Println("✅ Connected to database")
	fmt.Println()

	if c.SchemaFile != "" {
		// Migrate specific schema
		if err := client.ValidateSchema(c.SchemaFile); err != nil {
			return err
		}
		fmt.Printf("📦 Migrating: %s\n", c.SchemaFile)
		fmt.Println()
		return c.migrateSingleSchema(ctx, client, c.SchemaFile)
	}

	if !c.Interactive {
		// Non-interactive: migrate all
		return c.migrateAll(ctx, client)
	}

	// Interactive mode
	selection := c.promptSelection()

	if selection == "0" {
		fmt.Println("Exiting...")
		return nil
	}

	if selection == "" {
		return c.migrateAll(ctx, client)
	}

	return c.migrateSelected(ctx, client, selection)
}

// migrateSingleSchema migrates a single schema with error handling.
func (c *MigrateCmd) migrateSingleSchema(ctx context.Context, client *Client, schema string) error {
	if err := client.Migrate(ctx, schema); err != nil {
		return fmt.Errorf("failed to migrate %s: %w", schema, err)
	}
	fmt.Printf("✅ Successfully migrated: %s\n", schema)
	return nil
}

// migrateAll migrates all available schemas.
func (c *MigrateCmd) migrateAll(ctx context.Context, client *Client) error {
	fmt.Println("Migrating all schemas...")
	fmt.Println()

	for _, schema := range SchemaList {
		if err := c.migrateSchemaSilent(ctx, client, schema); err != nil {
			fmt.Printf("❌ Failed to migrate %s: %v\n", schema, err)
			return err
		}
	}

	fmt.Println()
	fmt.Println("============================================================================")
	fmt.Println("✅ Migration complete")
	fmt.Println("============================================================================")
	fmt.Println()

	return nil
}

// migrateSelected migrates schemas based on user selection.
func (c *MigrateCmd) migrateSelected(ctx context.Context, client *Client, selection string) error {
	selections := strings.Split(selection, ",")
	fmt.Println()

	for _, sel := range selections {
		sel = strings.TrimSpace(sel)
		idx, err := strconv.Atoi(sel)
		if err != nil || idx < 1 || idx > len(SchemaList) {
			fmt.Printf("⚠️  Invalid selection: %s\n", sel)
			continue
		}

		schema := SchemaList[idx-1]
		if err := c.migrateSchemaSilent(ctx, client, schema); err != nil {
			fmt.Printf("❌ Failed to migrate %s: %v\n", schema, err)
			return err
		}
	}

	fmt.Println()
	fmt.Println("============================================================================")
	fmt.Println("✅ Migration complete")
	fmt.Println("============================================================================")
	fmt.Println()

	return nil
}

// migrateSchemaSilent migrates a schema with minimal output.
func (c *MigrateCmd) migrateSchemaSilent(ctx context.Context, client *Client, schema string) error {
	fmt.Printf("📦 %s ... ", schema)
	if err := client.Migrate(ctx, schema); err != nil {
		fmt.Printf("❌\n   Error: %v\n", err)
		return err
	}
	fmt.Println("✅")
	return nil
}

// promptSelection displays the migration menu and reads user input.
func (c *MigrateCmd) promptSelection() string {
	fmt.Println("============================================================================")
	fmt.Println("NodeByte Schema Migration")
	fmt.Println("============================================================================")
	fmt.Println()
	fmt.Println("Available schemas:")
	fmt.Println()

	for i, schema := range SchemaList {
		fmt.Printf("  [%d] %s\n", i+1, schema)
	}
	fmt.Println("  [0] Exit")
	fmt.Println()
	fmt.Println("Which schema(s) would you like to migrate?")
	fmt.Println("Enter schema numbers (comma-separated) or press Enter to migrate all:")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
