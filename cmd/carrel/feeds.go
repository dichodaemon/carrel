package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/query"
)

func feedsCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "feeds <consumer>",
		Short: "List sources that feed a consumer",
		Long: `List all sources that feed configuration to a given consumer.

Examples:
  carrel feeds carrel
  carrel feeds carrel --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()
			q := query.New(reg)

			results, err := q.ListFeeds(args[0])
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			for _, r := range results {
				scope := "universal"
				if r.Scope == 1 {
					scope = "target-specific"
				}
				fmt.Printf("%-20s %-20s %s\n", r.Alias, scope, r.Path)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
