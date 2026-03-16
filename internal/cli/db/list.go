package db

import (
	"fmt"
)

// ListCmd lists all available database schemas.
type ListCmd struct{}

// Run executes the list command.
func (c *ListCmd) Run() error {
	fmt.Println("Available schemas:")
	fmt.Println()
	for i, schema := range SchemaList {
		fmt.Printf("  [%2d] %s\n", i+1, schema)
	}
	fmt.Println()
	return nil
}
