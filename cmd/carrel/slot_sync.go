package main

import (
	"fmt"
	"os"
	"github.com/google/uuid"
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

func slotSyncAllCmd() *cobra.Command {
	var dryRun bool
	var exclude []string

	cmd := &cobra.Command{
		Use:   "sync-all [consumer-alias]",
		Short: "Wire all unwired entries to slots",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			// Resolve consumer from arg or cwd
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

			// Resolve linked sources
			sources, err := reg.ResolveSources(c.ID)
			if err != nil {
				return fmt.Errorf("resolve sources: %w", err)
			}

			// Build source alias map and source ID list
			srcAliasByID := make(map[uuid.UUID]string)
			var sourceIDs []uuid.UUID
			for _, s := range sources {
				sourceIDs = append(sourceIDs, s.ID)
				srcAliasByID[s.ID] = s.Alias
			}

			// Get all entries from linked sources
			entries, err := reg.ResolveEntries(sourceIDs)
			if err != nil {
				return fmt.Errorf("resolve entries: %w", err)
			}

			// Get all slots for this consumer
			slots, err := reg.ResolveSlots(c.ID)
			if err != nil {
				return fmt.Errorf("resolve slots: %w", err)
			}

			// Build set of entry IDs linked to any slot
			linked := make(map[uuid.UUID]bool)
			for _, slot := range slots {
				slotEntries, err := reg.ResolveEntrySlots(slot.ID)
				if err != nil {
					return fmt.Errorf("resolve entry slots for %q: %w", slot.Name, err)
				}
				for _, se := range slotEntries {
					linked[se.ID] = true
				}
			}

			// Build exclude set
			excludeSet := make(map[string]bool)
			for _, ex := range exclude {
				excludeSet[ex] = true
			}

			// Compute unwired: entries not linked to any slot, excluding filtered ones
			var unwired []registry.Entry
			for _, e := range entries {
				if linked[e.ID] {
					continue
				}
				compound := fmt.Sprintf("%s:%s:%s", srcAliasByID[e.SourceID], typeNameStr(e.Type), e.Name)
				if excludeSet[compound] {
					continue
				}
				unwired = append(unwired, e)
			}

			// Dry run: print and exit
			if dryRun {
				if len(unwired) == 0 {
					fmt.Println("no unwired entries")
				} else {
					fmt.Printf("unwired entries (%d):\n", len(unwired))
					for _, e := range unwired {
						compound := fmt.Sprintf("%s:%s:%s", srcAliasByID[e.SourceID], typeNameStr(e.Type), e.Name)
						fmt.Printf("  %s\n", compound)
					}
				}
				return nil
			}

			// Build slot name → slot index
			slotByName := make(map[string]registry.Slot)
			for _, s := range slots {
				slotByName[s.Name] = s
			}

			wired := 0
			created := 0
			for _, e := range unwired {
				// Determine slot name, compose mode, and dest path
				var slotName string
				var composeMode registry.ComposeMode
				var destPath string

				if e.Type == registry.TypeAppendSystem {
					slotName = "APPEND_SYSTEM.md"
					composeMode = registry.ModeConcatenation
					destPath = "omp/APPEND_SYSTEM.md"
				} else {
					slotName = e.Name
					composeMode = registry.ModeOverride
					destPath = e.RelativePath
				}

				// Find or create slot
				slot, ok := slotByName[slotName]
				if !ok {
					cm := composeMode
					slot = registry.Slot{
						ID:          uuid.New(),
						ConsumerID:  c.ID,
						Name:        slotName,
						DestPath:    destPath,
						ComposeMode: &cm,
					}
					if err := reg.RegisterSlot(slot); err != nil {
						return fmt.Errorf("register slot %q: %w", slotName, err)
					}
					slotByName[slotName] = slot
					created++
				}

				// Link entry to slot
				if err := reg.LinkEntrySlot(e.ID, slot.ID, 0); err != nil {
					return fmt.Errorf("link entry %q to slot %q: %w", e.Name, slotName, err)
				}
				wired++
			}

			if created > 0 {
				fmt.Printf("created %d slot(s), ", created)
			}
			fmt.Printf("wired %d entries to slots for %s\n", wired, consumerAlias)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show unwired entries without modifying")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "Compound IDs to exclude (source:type:name)")

	return cmd
}
