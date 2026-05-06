package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/query"
)

func sourcesCmd() *cobra.Command {
	var typeFilter string
	var sourceFilter string
	var validOnly bool
	var orphanedOnly bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "sources",
		Short: "List source files with drift status",
		Long: `List all registered configuration entries with their on-disk status.

Each entry shows whether the file EXISTS or is MISSING (orphaned registry entry).
Filter by type, source, or status.

The last column is a compound identifier (SOURCE:TYPE:NAME) that can be
copied into 'carrel inspect' or 'carrel trace'.

Examples:
  carrel sources
  carrel sources --type=rule
  carrel sources --orphaned
  carrel sources --source=carrel-omp --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()
			q := query.New(reg)

			var opts query.FileQueryOpts
			if typeFilter != "" {
				typ, ok := lookupType(typeFilter)
				if !ok {
					return fmt.Errorf("unknown type %q", typeFilter)
				}
				opts.Type = &typ
			}
			if sourceFilter != "" {
				opts.Source = &sourceFilter
			}
			opts.Valid = validOnly
			opts.Orphaned = orphanedOnly

			results, err := q.ListFiles(opts)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			for _, r := range results {
				fmt.Printf("%-15s %-15s %-30s %-60s %-8s %s\n",
					r.SourceAlias, r.TypeName, r.Name, r.Path, r.Status, r.ID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by capability type")
	cmd.Flags().StringVar(&sourceFilter, "source", "", "Filter by source alias")
	cmd.Flags().BoolVar(&validOnly, "valid", false, "Show only existing files")
	cmd.Flags().BoolVar(&orphanedOnly, "orphaned", false, "Show only missing files")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
