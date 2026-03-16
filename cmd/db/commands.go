package main

import (
	"context"

	"github.com/spf13/cobra"

	dbcli "github.com/nodebyte/backend/internal/cli/db"
)

// InitCmd returns the init subcommand.
func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a fresh database with all schemas",
		Long:  "Initialize a fresh database by applying all schemas in order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseURL, _ := cmd.Flags().GetString("database")
			schemasDir, _ := cmd.Flags().GetString("schemas")

			initCmd, err := dbcli.NewInitCmd([]string{
				"-database", databaseURL,
				"-schemas", schemasDir,
			})
			if err != nil {
				return err
			}

			ctx := context.Background()
			return initCmd.Run(ctx)
		},
	}

	cmd.Flags().StringP("database", "d", "", "PostgreSQL connection string (or set DATABASE_URL env var)")
	cmd.Flags().String("schemas", "", "Path to schemas directory (optional)")

	return cmd
}

// MigrateCmd returns the migrate subcommand.
func MigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database schemas",
		Long:  "Apply database schemas either interactively or for a specific schema.",
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

	return cmd
}

// ResetCmd returns the reset subcommand.
func ResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset database (DROP and recreate)",
		Long:  "Reset database by dropping and recreating it, then applying all schemas. ⚠️  USE WITH CAUTION",
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseURL, _ := cmd.Flags().GetString("database")
			schemasDir, _ := cmd.Flags().GetString("schemas")

			resetCmd, err := dbcli.NewResetCmd([]string{
				"-database", databaseURL,
				"-schemas", schemasDir,
			})
			if err != nil {
				return err
			}

			ctx := context.Background()
			return resetCmd.Run(ctx)
		},
	}

	cmd.Flags().StringP("database", "d", "", "PostgreSQL connection string (or set DATABASE_URL env var)")
	cmd.Flags().String("schemas", "", "Path to schemas directory (optional)")

	return cmd
}

// ListCmd returns the list subcommand.
func ListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available schemas",
		RunE: func(cmd *cobra.Command, args []string) error {
			return (&dbcli.ListCmd{}).Run()
		},
	}

	return cmd
}
