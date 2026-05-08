package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/query"
)

func planCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "plan [consumer]",
		Short: "Preview what the next deployment would produce",
		Long: `Shows what the next 'carrel run' would deploy compared to the last deployment.
Runs the full compose pipeline in-memory. No files are written.

If no consumer is specified, plans for the consumer resolved from cwd.

Examples:
  carrel plan
  carrel plan carrel
  carrel plan --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var consumerAlias string
			if len(args) > 0 {
				consumerAlias = args[0]
			} else {
				cwd, _ := os.Getwd()
				gitRoot := findGitRoot(cwd)
				if gitRoot == "" {
					return fmt.Errorf("not in a git repository; specify consumer or run from a repo")
				}
				reg := mustOpenRegistry()
				c, err := reg.ResolveConsumer(gitRoot)
				reg.Close()
				if err != nil {
					return fmt.Errorf("unregistered repo; specify consumer explicitly")
				}
				consumerAlias = c.Alias
			}

			reg := mustOpenRegistry()
			defer reg.Close()
			q := query.New(reg)

			results, err := q.Plan(consumerAlias)
			if err != nil {
				return err
			}

			// Check slots.yml consistency
			for _, w := range checkSlotsConsistency(reg, consumerAlias) {
				fmt.Fprintf(os.Stderr, "warning: slots.yml: %s\n", w)
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			for _, r := range results {
				fmt.Printf("%-10s %-15s %s\n", r.Change, r.ConsumerAlias, r.Path)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
