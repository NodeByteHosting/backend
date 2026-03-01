// Package main provides a CLI for database schema migrations.
// This is a convenience wrapper around the db migrate command.
// For more operations, use: db init|migrate|reset|list
package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	dbcli "github.com/nodebyte/backend/internal/cli/db"
)

func main() {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database schemas",
		Long:  "Apply database schemas interactively or migrate a specific schema.",
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseURL, _ := cmd.Flags().GetString("database")
			schemaFile, _ := cmd.Flags().GetString("schema")
			schemasDir, _ := cmd.Flags().GetString("schemas")

			migrateCmd, err := dbcli.NewMigrateCmd([]string{
				"-database", databaseURL,
				"-schema", schemaFile,
				"-schemas", schemasDir,
			})
			if err != nil {
				return err
			}

			ctx := context.Background()
			return migrateCmd.Run(ctx)
		},
	}

	cmd.Flags().StringP("database", "d", "", "PostgreSQL connection string (or set DATABASE_URL env var)")
	cmd.Flags().String("schema", "", "Specific schema file to migrate (optional)")
	cmd.Flags().String("schemas", "", "Path to schemas directory (optional)")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
