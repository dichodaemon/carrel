package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/query"
)

func previewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview <consumer>:<path>",
		Short: "Preview the composed content of a deployment file without writing to disk",
		Long: `Preview what a deployment file would contain after the next 'carrel run'.

Runs the full compose pipeline in-memory and displays the resolved content
for a single file. No files are written.

The identifier format is <consumer-alias>:<absolute-path>, matching the
deployed file path. This is the same format as 'carrel inspect' for deployed files.

Examples:
  carrel preview carrel:/workspace/carrel/.omp/APPEND_SYSTEM.md
  carrel preview carrel:.omp/APPEND_SYSTEM.md   (relative path resolved against consumer root)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			reg := mustOpenRegistry()
			defer reg.Close()
			q := query.New(reg)

			parts := strings.SplitN(id, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid identifier format; expected <consumer>:<path> (got %q)", id)
			}
			consumerAlias := parts[0]
			pathArg := parts[1]

			// If the path is relative, resolve against the consumer root.
			// Absolute paths are used as-is.
			if !filepath.IsAbs(pathArg) {
				c, err := reg.ResolveConsumer(consumerAlias)
				if err != nil {
					return fmt.Errorf("consumer %q: %w", consumerAlias, err)
				}
				pathArg = filepath.Join(c.DeployRoot, pathArg)
			}

			pf, err := q.Preview(consumerAlias, pathArg)
			if err != nil {
				return err
			}

			fmt.Printf("--- %s ---\n", pf.Path)
			fmt.Print(string(pf.Content))
			return nil
		},
	}

	return cmd
}
