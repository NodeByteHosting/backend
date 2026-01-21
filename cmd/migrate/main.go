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
	databaseURL := flag.String("database", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	schemaFile := flag.String("schema", "", "Specific schema file to migrate (optional)")
	flag.Parse()

	if *databaseURL == "" {
		fmt.Println("❌ Error: No database URL provided")
		fmt.Println("Usage: migrate -database <DATABASE_URL> [-schema <schema_file>]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  migrate -database \"postgresql://user:password@localhost:5432/nodebyte\"")
		fmt.Println("  migrate -database \"postgresql://user:password@localhost:5432/nodebyte\" -schema schema_01_users_auth.sql")
		fmt.Println()
		fmt.Println("Environment variable: DATABASE_URL")
		os.Exit(1)
	}

	// Get schemas directory
	schemasDir := filepath.Join(filepath.Dir(os.Args[0]), "..", "..", "schemas")
	if info, err := os.Stat(schemasDir); err != nil || !info.IsDir() {
		// Try relative to current working directory
		schemasDir = "schemas"
		if info, err := os.Stat(schemasDir); err != nil || !info.IsDir() {
			fmt.Println("❌ Error: schemas directory not found")
			os.Exit(1)
		}
	}

	// Connect to database
	conn, err := pgx.Connect(context.Background(), *databaseURL)
	if err != nil {
		fmt.Printf("❌ Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	fmt.Println("✅ Connected to database")

	if *schemaFile != "" {
		// Migrate specific schema
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

	fmt.Printf("📦 Migrating: %s\n", schemaFile)
	fmt.Println()

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
		fmt.Println("   Run manually to see full error details")
		return
	}

	fmt.Println("✅")
}
