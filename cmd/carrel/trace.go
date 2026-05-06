package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/query"
)

func traceCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "trace <id>",
		Short: "Trace composition provenance",
		Long: `Trace the relationship between source files and deployed files.

For a source file (<source>:<type>:<name>), shows all consumers that deployed it.
For a deployed file (<consumer>:<path>), shows all source files that contributed.

Examples:
  carrel trace carrel-omp:rule:no-push-oh-my-pi
  carrel trace carrel:/workspace/carrel/.omp/rules/no-push-oh-my-pi.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			reg := mustOpenRegistry()
			defer reg.Close()
			q := query.New(reg)

			parts := strings.SplitN(id, ":", 3)
			if len(parts) == 3 {
				// Source file: <source>:<type>:<name>
				sourceAlias := parts[0]
				typeName := parts[1]
				name := parts[2]
				typ, ok := lookupType(typeName)
				if !ok {
					return fmt.Errorf("unknown type %q", typeName)
				}

				results, err := q.TraceSource(sourceAlias, typ, name)
				if err != nil {
					return err
				}

				if jsonOut {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(results)
				}

				fmt.Printf("Source: %s:%s:%s\n\n", sourceAlias, typeName, name)
				if len(results) == 0 {
					fmt.Println("Not deployed anywhere.")
					return nil
				}
				fmt.Println("Deployed to:")
				for _, r := range results {
					fmt.Printf("  %-20s %s\n", r.ConsumerAlias, r.DeployedPath)
				}
				return nil
			}

			if len(parts) == 2 {
				// Check if it looks like a malformed 3-part source ID
				reg2 := mustOpenRegistry()
				sources, _ := reg2.ListSources()
				reg2.Close()
				for _, s := range sources {
					if s.Alias == parts[0] {
						return fmt.Errorf("source identifiers require three parts: <source>:<type>:<name> (e.g., %s:rule:%s)", parts[0], parts[1])
					}
				}

				// Deployed file: <consumer>:<path>
				consumerAlias := parts[0]
				path := parts[1]

				results, err := q.TraceDeployed(consumerAlias, path)
				if err != nil {
					return err
				}

				if jsonOut {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(results)
				}

				fmt.Printf("Deployed file: %s:%s\n\n", consumerAlias, path)
				if len(results) == 0 {
					fmt.Println("No source entries found.")
					return nil
				}
				fmt.Println("Source entries:")
				for _, r := range results {
					fmt.Printf("  %-20s %-15s %-20s %s\n", r.SourceAlias, typeNameStr(r.SourceType), r.SourceName, r.SourcePath)
				}
				return nil
			}

			return fmt.Errorf("invalid identifier format; expected <source>:<type>:<name> or <consumer>:<path>")
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
