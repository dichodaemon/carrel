package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/registry"
	"github.com/dichodaemon/carrel/internal/scanner"
)

func localCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Manage local per-machine configuration overrides",
		Long:  "Commands for managing a local configuration source (~/.carrel/local/) that provides per-user or per-host overrides layered on top of universal configuration.",
	}

	cmd.AddCommand(localInitCmd())
	cmd.AddCommand(localScanCmd())
	cmd.AddCommand(localPathCmd())

	return cmd
}

func localInitCmd() *cobra.Command {
	var hostScope bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the local configuration source",
		Long:  "Creates ~/.carrel/local/ and registers it as a configuration source. Idempotent — skips if already registered.",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("home dir: %w", err)
			}
			localPath := filepath.Join(home, ".carrel", "local")

			if err := os.MkdirAll(localPath, 0755); err != nil {
				return fmt.Errorf("create %s: %w", localPath, err)
			}

			reg := mustOpenRegistry()
			defer reg.Close()

			// Check if already registered
			sources, _ := reg.ListSources()
			for _, s := range sources {
				if s.Path == localPath {
					fmt.Printf("local source already registered: %s\n", localPath)
					return nil
				}
			}

			scope := registry.ScopeUserPersonal
			if hostScope {
				scope = registry.ScopeHostLocal
			}

			src := registry.Source{
				ID:    uuid.New(),
				Alias: "local",
				Path:  localPath,
				Scope: scope,
				Kind:  registry.SourceHostLocal,
			}
			if err := reg.RegisterSource(src); err != nil {
				return fmt.Errorf("register local source: %w", err)
			}

			fmt.Printf("local source registered: %s (scope: %s)\n", localPath, scopeStr(scope))
			return nil
		},
	}

	cmd.Flags().BoolVar(&hostScope, "host", false, "Register as host-local scope instead of user-personal")
	return cmd
}

func localScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan local source and link entries to existing slots",
		Long:  "Scans the local configuration source (~/.carrel/local/) for convention-matching files, registers them as entries, and links them to existing slots with scope-derived priority.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			// Find the local source
			sources, err := reg.ListSources()
			if err != nil {
				return err
			}
			var localSrc *registry.Source
			for _, s := range sources {
				if s.Scope == registry.ScopeUserPersonal || s.Scope == registry.ScopeHostLocal {
					localSrc = &s
					break
				}
			}
			if localSrc == nil {
				return fmt.Errorf("no local source found; run 'carrel local init' first")
			}

			// Scan the source
			results, err := scanner.ScanSource(reg, *localSrc)
			if err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			for _, r := range results {
				if r.Action == "error" {
					fmt.Printf("%-12s %-15s %-20s error: %v\n", r.Action, r.Type, r.Name, r.Error)
				} else {
					fmt.Printf("%-12s %-15s %s\n", r.Action, r.Type, r.Name)
				}
			}

			// Link scanned entries to existing slots with scope-derived priority
			priority := registry.ScopePriority[localSrc.Scope]
			if err := linkLocalEntries(reg, *localSrc, priority); err != nil {
				return fmt.Errorf("link entries to slots: %w", err)
			}

			return nil
		},
	}
}

func localPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the local source directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("home dir: %w", err)
			}
			fmt.Println(filepath.Join(home, ".carrel", "local"))
			return nil
		},
	}
}

// linkLocalEntries links local-source entries to existing slots with the given priority.
// Only links entries whose name matches an existing slot name (per consumer).
func linkLocalEntries(reg registry.Registry, source registry.Source, priority int) error {
	sources, _ := reg.ListSources()
	var sourceIDs []uuid.UUID
	for _, s := range sources {
		sourceIDs = append(sourceIDs, s.ID)
	}
	entries, _ := reg.ResolveEntries(sourceIDs)

	// Filter to only this source's entries
	var localEntries []registry.Entry
	for _, e := range entries {
		if e.SourceID == source.ID {
			localEntries = append(localEntries, e)
		}
	}

	consumers, _ := reg.ListConsumers()
	for _, c := range consumers {
		slots, _ := reg.ResolveSlots(c.ID)
		for _, slot := range slots {
			for _, e := range localEntries {
				if e.Name == slot.Name {
					_ = reg.LinkEntrySlot(e.ID, slot.ID, priority)
				}
			}
		}
	}

	return nil
}
