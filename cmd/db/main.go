package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

var schemas = []string{
	"schema_01_users_auth.sql",
	"schema_02_pterodactyl_sync.sql",
	"schema_03_servers.sql",
	"schema_04_billing.sql",
	"schema_05_support_tickets.sql",
	"schema_06_discord_webhooks.sql",
	"schema_07_sync_logs.sql",
	"schema_08_config.sql",
	"schema_09_hytale.sql",
	"schema_10_hytale_audit.sql",
	"schema_11_hytale_server_logs.sql",
	"schema_12_server_subusers.sql",
	"schema_13_hytale_server_link.sql",
	"schema_14_partners.sql",
	"schema_15_careers.sql",
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		cmdInit(args)
	case "migrate":
		cmdMigrate(args)
	case "reset":
		cmdReset(args)
	case "list":
		cmdList()
	case "help":
		printUsage()
	default:
		fmt.Printf("❌ Unknown command: %s\n", command)
		fmt.Println()
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`NodeByte Database Tool

Usage: db <command> [options]

Commands:
  init       Initialize a fresh database with all schemas
  migrate    Migrate specific or all schemas to an existing database
  reset      Reset database (DROP and recreate) - USE WITH CAUTION
  list       List all available schemas
  help       Show this help message

Examples:
  # Initialize fresh database
  db init -database "postgresql://user:pass@localhost/nodebyte"

  # Migrate all schemas
  db migrate -database "postgresql://user:pass@localhost/nodebyte"

  # Migrate specific schema
  db migrate -database "postgresql://user:pass@localhost/nodebyte" -schema schema_14_partners.sql

  # Reset database (destructive - requires confirmation)
  db reset -database "postgresql://user:pass@localhost/nodebyte"

Environment Variables:
  DATABASE_URL - Default database connection string
`)
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	databaseURL := fs.String("database", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	fs.Parse(args)

	if *databaseURL == "" {
		fmt.Println("❌ Error: No database URL provided")
		fmt.Println("Use: db init -database <DATABASE_URL>")
		os.Exit(1)
	}

	fmt.Println("============================================================================")
	fmt.Println("NodeByte Database Initialization")
	fmt.Println("============================================================================")
	fmt.Println()

	conn, err := pgx.Connect(context.Background(), *databaseURL)
	if err != nil {
		fmt.Printf("❌ Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	fmt.Println("✅ Connected to database")
	fmt.Println()
	fmt.Printf("📦 Initializing database with %d schemas...\n", len(schemas))
	fmt.Println()

	schemasDir := findSchemasDir()
	if schemasDir == "" {
		fmt.Println("❌ Error: schemas directory not found")
		os.Exit(1)
	}

	for _, schema := range schemas {
		if err := migrateSchema(conn, schemasDir, schema); err != nil {
			fmt.Printf("❌ %s\n   Error: %v\n", schema, err)
			os.Exit(1)
		}
		fmt.Printf("✅ %s\n", schema)
	}

	fmt.Println()
	fmt.Println("============================================================================")
	fmt.Println("✅ Database initialization complete!")
	fmt.Println("============================================================================")
	fmt.Println()
}

func cmdMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	databaseURL := fs.String("database", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	schemaFile := fs.String("schema", "", "Specific schema file to migrate (optional)")
	fs.Parse(args)

	if *databaseURL == "" {
		fmt.Println("❌ Error: No database URL provided")
		fmt.Println("Use: db migrate -database <DATABASE_URL> [-schema <schema_file>]")
		os.Exit(1)
	}

	conn, err := pgx.Connect(context.Background(), *databaseURL)
	if err != nil {
		fmt.Printf("❌ Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	fmt.Println("✅ Connected to database")
	fmt.Println()

	schemasDir := findSchemasDir()
	if schemasDir == "" {
		fmt.Println("❌ Error: schemas directory not found")
		os.Exit(1)
	}

	if *schemaFile != "" {
		// Migrate specific schema
		fmt.Printf("📦 Migrating: %s\n", *schemaFile)
		fmt.Println()
		if err := migrateSchema(conn, schemasDir, *schemaFile); err != nil {
			fmt.Printf("❌ Failed to migrate: %s\n", *schemaFile)
			fmt.Printf("   Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Successfully migrated: %s\n", *schemaFile)
	} else {
		// Interactive menu
		printMenu()
		selection := readSelection()

		if selection == "0" {
			fmt.Println("Exiting...")
			return
		}

		if selection == "" {
			// Migrate all
			fmt.Println("Migrating all schemas...")
			fmt.Println()
			migrateAll(conn, schemasDir)
		} else {
			// Migrate selected
			migrateSelected(conn, schemasDir, selection)
		}

		fmt.Println()
		fmt.Println("============================================================================")
		fmt.Println("✅ Migration complete")
		fmt.Println("============================================================================")
		fmt.Println()
	}
}

func cmdReset(args []string) {
	fs := flag.NewFlagSet("reset", flag.ExitOnError)
	databaseURL := fs.String("database", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	fs.Parse(args)

	if *databaseURL == "" {
		fmt.Println("❌ Error: No database URL provided")
		fmt.Println("Use: db reset -database <DATABASE_URL>")
		os.Exit(1)
	}

	// Parse database name from connection string
	connConfig, err := pgx.ParseConfig(*databaseURL)
	if err != nil {
		fmt.Printf("❌ Error parsing database URL: %v\n", err)
		os.Exit(1)
	}

	dbName := connConfig.Database

	// Confirm before resetting
	fmt.Printf("⚠️  WARNING: This will DROP and recreate the database '%s'\n", dbName)
	fmt.Print("Are you SURE? Type the database name to confirm: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != dbName {
		fmt.Println("❌ Confirmation failed. Database not reset.")
		os.Exit(1)
	}

	// Connect to postgres (not the target database)
	pgConfig := *connConfig
	pgConfig.Database = "postgres"
	conn, err := pgx.ConnectConfig(context.Background(), &pgConfig)
	if err != nil {
		fmt.Printf("❌ Error connecting to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	fmt.Println()
	fmt.Printf("🔄 Dropping database '%s'...\n", dbName)
	_, err = conn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
	if err != nil {
		fmt.Printf("❌ Error dropping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔄 Creating database '%s'...\n", dbName)
	_, err = conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s;", dbName))
	if err != nil {
		fmt.Printf("❌ Error creating database: %v\n", err)
		os.Exit(1)
	}

	// Now connect to the new database and initialize
	newConn, err := pgx.Connect(context.Background(), *databaseURL)
	if err != nil {
		fmt.Printf("❌ Error connecting to new database: %v\n", err)
		os.Exit(1)
	}
	defer newConn.Close(context.Background())

	fmt.Println()
	fmt.Printf("📦 Initializing database with %d schemas...\n", len(schemas))
	fmt.Println()

	schemasDir := findSchemasDir()
	if schemasDir == "" {
		fmt.Println("❌ Error: schemas directory not found")
		os.Exit(1)
	}

	for _, schema := range schemas {
		if err := migrateSchema(newConn, schemasDir, schema); err != nil {
			fmt.Printf("❌ %s\n   Error: %v\n", schema, err)
			os.Exit(1)
		}
		fmt.Printf("✅ %s\n", schema)
	}

	fmt.Println()
	fmt.Println("============================================================================")
	fmt.Println("✅ Database reset and initialization complete!")
	fmt.Println("============================================================================")
	fmt.Println()
}

func cmdList() {
	fmt.Println("Available schemas:")
	fmt.Println()
	for i, schema := range schemas {
		fmt.Printf("  [%2d] %s\n", i+1, schema)
	}
	fmt.Println()
}

func printMenu() {
	fmt.Println("============================================================================")
	fmt.Println("NodeByte Schema Migration")
	fmt.Println("============================================================================")
	fmt.Println()
	fmt.Println("Available schemas:")
	fmt.Println()

	for i, schema := range schemas {
		fmt.Printf("  [%d] %s\n", i+1, schema)
	}
	fmt.Println("  [0] Exit")
	fmt.Println()
	fmt.Println("Which schema(s) would you like to migrate?")
	fmt.Println("Enter schema numbers (comma-separated) or press Enter to migrate all:")
}

func readSelection() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func migrateSchema(conn *pgx.Conn, schemasDir, schemaFile string) error {
	filePath := filepath.Join(schemasDir, schemaFile)

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("schema file not found: %s", filePath)
	}

	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	_, err = conn.Exec(context.Background(), string(sqlBytes))
	if err != nil {
		return err
	}

	return nil
}

func migrateAll(conn *pgx.Conn, schemasDir string) {
	for _, schema := range schemas {
		migrateSingleQuiet(conn, schemasDir, schema)
	}
}

func migrateSelected(conn *pgx.Conn, schemasDir, selection string) {
	selections := strings.Split(selection, ",")

	for _, sel := range selections {
		sel = strings.TrimSpace(sel)
		idx, err := strconv.Atoi(sel)
		if err != nil || idx < 1 || idx > len(schemas) {
			fmt.Printf("⚠️  Invalid selection: %s\n", sel)
			continue
		}

		schema := schemas[idx-1]
		migrateSingleQuiet(conn, schemasDir, schema)
	}
}

func migrateSingleQuiet(conn *pgx.Conn, schemasDir, schema string) {
	filePath := filepath.Join(schemasDir, schema)

	if _, err := os.Stat(filePath); err != nil {
		fmt.Printf("❌ Schema file not found: %s\n", filePath)
		return
	}

	fmt.Printf("📦 %s ... ", schema)

	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("❌\n   Error: %v\n", err)
		return
	}

	_, err = conn.Exec(context.Background(), string(sqlBytes))
	if err != nil {
		fmt.Printf("❌\n   Error: %v\n", err)
		return
	}

	fmt.Println("✅")
}

func findSchemasDir() string {
	// Try different possible locations
	possiblePaths := []string{
		"./schemas",         // Current directory
		"./backend/schemas", // Root directory
		"../schemas",        // Up one directory
		filepath.Join(os.Getenv("PWD"), "schemas"),
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}

	return ""
}
