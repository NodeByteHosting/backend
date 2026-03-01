package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "db",
		Short: "NodeByte Database Management Tool",
		Long:  "Database initialization, migration, and management utility for NodeByte backend.",
	}

	// Add subcommands
	rootCmd.AddCommand(InitCmd())
	rootCmd.AddCommand(MigrateCmd())
	rootCmd.AddCommand(ResetCmd())
	rootCmd.AddCommand(ListCmd())

	// Add global flags
	rootCmd.PersistentFlags().StringP("database", "d", "", "PostgreSQL connection string (or set DATABASE_URL env var)")
	rootCmd.PersistentFlags().String("schemas", "", "Path to schemas directory (optional)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
