package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/google/uuid"
	"github.com/dichodaemon/carrel/internal/query"
	"github.com/dichodaemon/carrel/internal/registry"
)

func inspectCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a source file or deployed file by compound identifier",
		Long: `Inspect a configuration entry by its compound identifier.

Dispatches to the appropriate detail view based on the identifier format:
  <source-alias>:<type>:<name>  → shows metadata and file content
  <consumer-alias>:<path>      → shows claim metadata and file content

Examples:
  carrel inspect carrel-omp:rule:no-push-oh-my-pi
  carrel inspect carrel:/workspace/carrel/.omp/rules/no-push-oh-my-pi.md`,
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

				// Show metadata
			sources, _ := reg.ListSources()
			var sourceIDs []uuid.UUID
			for _, s := range sources {
					sourceIDs = append(sourceIDs, s.ID)
				}
				entries, _ := reg.ResolveEntries(sourceIDs)
				for _, e := range entries {
					if e.Type == typ && e.Name == name {
						// Read content
						for _, s := range sources {
							data, err := os.ReadFile(filepath.Join(s.Path, e.RelativePath))
							if err == nil {
								fmt.Println("--- Content ---")
								fmt.Println(string(data))
								break
							}
						}
					}
				}

				if len(results) > 0 {
					fmt.Println("\nDeployed to:")
					for _, r := range results {
						fmt.Printf("  %s: %s\n", r.ConsumerAlias, r.DeployedPath)
					}
				}
				return nil
			}

			if len(parts) == 2 {
				reg2 := mustOpenRegistry()
				sources, _ := reg2.ListSources()
				reg2.Close()
				for _, s := range sources {
					if s.Alias == parts[0] {
						return fmt.Errorf("source identifiers require three parts: <source>:<type>:<name> (got 2 parts: %q)", strings.Join(parts, ":"))
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

				// Read deployed file content
				data, err := os.ReadFile(path)
				if err == nil {
					fmt.Println("--- Content ---")
					fmt.Println(string(data))
				}

				if len(results) > 0 {
					fmt.Println("\nSource entries:")
					for _, r := range results {
						fmt.Printf("  %s:%s:%s → %s\n", r.SourceAlias, typeNameStr(r.SourceType), r.SourceName, r.SourcePath)
					}
				}
				return nil
			}

			return fmt.Errorf("invalid identifier format; expected <source>:<type>:<name> or <consumer>:<path>")
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func typeNameStr(typ registry.CapabilityType) string {
	names := map[registry.CapabilityType]string{
		registry.TypeRule:          "rule",
		registry.TypeSkill:         "skill",
		registry.TypeCommand:       "command",
		registry.TypeExtension:     "extension",
		registry.TypeAgent:         "agent",
		registry.TypeTool:          "tool",
		registry.TypeHook:          "hook",
		registry.TypePrompt:        "prompt",
		registry.TypeInstruction:   "instruction",
		registry.TypeContextFile:   "context-file",
		registry.TypeAppendSystem:  "append-system",
		registry.TypeZshConfig:     "zsh",
		registry.TypeNvimConfig:    "nvim",
		registry.TypeWeztermConfig: "wezterm",
		registry.TypeP10kConfig:    "p10k",
		registry.TypeHelixConfig:   "helix",
	}
	if n, ok := names[typ]; ok {
		return n
	}
	return fmt.Sprintf("type-%d", typ)
}
