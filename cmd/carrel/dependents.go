package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/query"
)

func dependentsCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "dependents <source>",
		Short: "List consumers that depend on a source",
		Long: `List all consumers that include entries from a given source.

Examples:
  carrel dependents carrel-omp
  carrel dependents carrel-omp --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()
			q := query.New(reg)

			results, err := q.ListDependents(args[0])
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			for _, r := range results {
				fmt.Printf("%-20s %s\n", r.Alias, r.Path)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
