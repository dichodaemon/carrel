package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/registry"
	"github.com/dichodaemon/carrel/internal/scanner"
)

func slotSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [consumer-alias]",
		Short: "Sync slot declarations to the registry database",
		Long: `Re-apply slot declarations from .carrel/slots.yml or source slot-defaults.yml
to the registry database. Use after pulling updates to pick up new slots
or entry links.

Resolves the consumer from cwd if no alias is given.

When the consumer has .carrel/slots.yml: applies that file.
Otherwise: applies slot-defaults.yml from each universal source.

Examples:
  carrel slot sync
  carrel slot sync folio`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			var consumerAlias string
			if len(args) > 0 {
				consumerAlias = args[0]
			} else {
				cwd, _ := os.Getwd()
				gitRoot := findGitRoot(cwd)
				if gitRoot == "" {
					return fmt.Errorf("not in a git repository; specify consumer or run from a repo")
				}
				c, err := reg.ResolveConsumer(gitRoot)
				if err != nil {
					return fmt.Errorf("unregistered repo; specify consumer explicitly")
				}
				consumerAlias = c.Alias
			}

			c, err := reg.ResolveConsumer(consumerAlias)
			if err != nil {
				return fmt.Errorf("consumer %q: %w", consumerAlias, err)
			}

			// Scan linked sources to register new entries from remote
			srcList, err := reg.ResolveSources(c.ID)
			if err != nil {
				return fmt.Errorf("resolve sources: %w", err)
			}

			allSources, _ := reg.ListSources()
			srcByID := make(map[string]registry.Source)
			for _, s := range allSources {
				srcByID[s.ID.String()] = s
			}

			for _, s := range srcList {
				src, ok := srcByID[s.ID.String()]
				if !ok {
					continue
				}
				results, scanErr := scanner.ScanSource(reg, src)
				if scanErr != nil {
					fmt.Printf("warning: scan %s: %v\n", src.Alias, scanErr)
					continue
				}
				for _, r := range results {
					if r.Action != "skipped" {
						fmt.Printf("  %-12s %-15s %s\n", r.Action, r.Type, r.Name)
					}
				}
			}

			consumerSlotsPath := filepath.Join(c.Path, ".carrel", "slots.yml")
			if _, statErr := os.Stat(consumerSlotsPath); statErr == nil {
				// Consumer has its own slots.yml — apply it
				warnings, err := registry.ApplySlotsFile(reg, c, consumerSlotsPath)
				if err != nil {
					return fmt.Errorf("apply slots.yml: %w", err)
				}
				for _, w := range warnings {
					fmt.Printf("warning: %s\n", w)
				}

				fmt.Printf("synced slots.yml for %s\n", consumerAlias)
				return nil
			}

			// No consumer slots.yml — apply source defaults
			sources, err := reg.ResolveSources(c.ID)
			if err != nil {
				return fmt.Errorf("resolve sources: %w", err)
			}

			applied := 0
			for _, s := range sources {
				defaultsPath := filepath.Join(s.Path, "slot-defaults.yml")
				warnings, err := registry.ApplySlotDefaults(reg, c, s, defaultsPath)
				if err != nil {
					if !os.IsNotExist(err) {
						fmt.Printf("warning: %s: %v\n", s.Alias, err)
					}
					continue
				}
				for _, w := range warnings {
					fmt.Printf("warning: %s: %s\n", s.Alias, w)
				}
				if len(warnings) == 0 {
					applied++
				}
			}

			if applied > 0 {
				fmt.Printf("synced slot-defaults.yml from %d source(s) for %s\n", applied, consumerAlias)
			} else {
				fmt.Printf("no slot-defaults.yml found for %s\n", consumerAlias)
			}
			return nil
		},
	}

	return cmd
}
