package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/query"
)

func deployedCmd() *cobra.Command {
	var consumerFilter string
	var claimsOnly bool
	var actualOnly bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "deployed",
		Short: "List deployed files with verification status",
		Long: `List files deployed to consumers with verification against disk.

Status: DEPLOYED (hash matches claim), MODIFIED (hash changed),
MISSING (file absent), FOREIGN (file on disk but not in claim).

The last column is a compound identifier (CONSUMER:PATH) that can be
copied into 'carrel inspect' or 'carrel trace'.

Examples:
  carrel deployed
  carrel deployed --consumer=carrel
  carrel deployed --claims
  carrel deployed --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()
			q := query.New(reg)

			var opts query.DeployQueryOpts
			if consumerFilter != "" {
				opts.Consumer = &consumerFilter
			}
			opts.Claims = claimsOnly
			opts.Actual = actualOnly

			results, err := q.ListDeployed(opts)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			for _, r := range results {
			fmt.Printf("%-15s %-60s %-10s %s\n", r.ConsumerAlias, r.Path, r.Status, r.ID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&consumerFilter, "consumer", "", "Filter by consumer alias")
	cmd.Flags().BoolVar(&claimsOnly, "claims", false, "Show registry claims only (no disk check)")
	cmd.Flags().BoolVar(&actualOnly, "actual", false, "Show filesystem only (no claim check)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
