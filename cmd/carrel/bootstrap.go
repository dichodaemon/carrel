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

func runBootstrap(cmd *cobra.Command, args []string) error {
	registryDir := "/workspace/.carrel"

	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return fmt.Errorf("bootstrap: create registry dir: %w", err)
	}

	reg, err := registry.New(registryDir)
	if err != nil {
		return fmt.Errorf("bootstrap: open registry: %w", err)
	}
	defer reg.Close()

	carrelConsumer := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "carrel",
		Path:       "/workspace/carrel",
		DeployRoot: "/workspace/carrel/.omp",
		Kind:       registry.ConsumerRepo,
	}
	_ = reg.RegisterConsumer(carrelConsumer)

	folioConsumer := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "folio",
		Path:       "/workspace/folio",
		DeployRoot: "/workspace/folio/.omp",
		Kind:       registry.ConsumerRepo,
	}
	_ = reg.RegisterConsumer(folioConsumer)

	containerConsumer := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "container",
		Path:       "/home/dev",
		DeployRoot: "/home/dev",
		Kind:       registry.ConsumerContainer,
	}
	_ = reg.RegisterConsumer(containerConsumer)
	hostConsumer := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "host",
		Path:       "/home/dev",
		DeployRoot: "/home/dev",
		Kind:       registry.ConsumerHost,
	}
	_ = reg.RegisterConsumer(hostConsumer)

	carrelSource := registry.Source{
		ID:    uuid.New(),
		Alias: "carrel-omp",
		Path:  "/workspace/carrel/omp",
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	_ = reg.RegisterSource(carrelSource)

	// Resolve actual consumer IDs (handles idempotent re-runs)
	carrelC, err := reg.ResolveConsumer("carrel")
	if err != nil {
		return fmt.Errorf("bootstrap: resolve carrel consumer: %w", err)
	}
	folioC, err := reg.ResolveConsumer("folio")
	if err != nil {
		return fmt.Errorf("bootstrap: resolve folio consumer: %w", err)
	}

	// List sources to get actual ID
	sources, err := reg.ListSources()
	if err != nil {
		return fmt.Errorf("bootstrap: list sources: %w", err)
	}
	var csID uuid.UUID
	for _, s := range sources {
		if s.Alias == "carrel-omp" {
			csID = s.ID
			break
		}
	}

	// Link universal source to consumers
	_ = reg.LinkConsumerSource(carrelC.ID, csID)
	_ = reg.LinkConsumerSource(folioC.ID, csID)

	// Scan source directory and auto-register entries
	scanSource := registry.Source{
		ID:    csID,
		Alias: "carrel-omp",
		Path:  "/workspace/carrel/omp",
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	results, err := scanner.ScanSource(reg, scanSource)
	if err != nil {
		return fmt.Errorf("bootstrap: scan source: %w", err)
	}
	for _, r := range results {
		if r.Action == "error" {
			fmt.Printf("  %-12s %-15s %-20s error: %v\n", r.Action, r.Type, r.Name, r.Error)
		} else {
			fmt.Printf("  %-12s %-15s %s\n", r.Action, r.Type, r.Name)
		}
	}

	// Generate default slots for each consumer
	containerC, _ := reg.ResolveConsumer("container")
	hostC, _ := reg.ResolveConsumer("host")
	for _, c := range []registry.Consumer{carrelC, folioC, containerC, hostC} {
		slotsPath := filepath.Join(c.Path, ".carrel", "slots.yml")
		warnings, err := registry.ApplySlotsFile(reg, c, slotsPath)
		if err == nil && warnings == nil || (err == nil && len(warnings) > 0) {
			// File existed and was applied (with or without warnings)
			for _, w := range warnings {
				fmt.Printf("  bootstrap: slots warning for %s: %s\n", c.Alias, w)
			}
		} else {
			// File doesn't exist or couldn't be read — fall back to convention
			if err := generateDefaultSlots(reg, c); err != nil {
				fmt.Printf("  bootstrap: slots for %s: %v\n", c.Alias, err)
			}
		}
	}

	fmt.Println("bootstrap: registry initialized")
	return nil
}

func generateDefaultSlots(reg registry.Registry, c registry.Consumer) error {
	isRepo := c.Kind == registry.ConsumerRepo
	isContainer := c.Kind == registry.ConsumerContainer || c.Kind == registry.ConsumerHost

	// Build source scope lookup
	sources, _ := reg.ListSources()
	sourceScope := make(map[uuid.UUID]registry.SourceScope)
	for _, s := range sources {
		sourceScope[s.ID] = s.Scope
	}

	var sourceIDs []uuid.UUID
	for _, s := range sources {
		sourceIDs = append(sourceIDs, s.ID)
	}
	entries, _ := reg.ResolveEntries(sourceIDs)

	for typ, conv := range registry.Conventions {
		if isRepo && isOSType(typ) {
			continue
		}
		if isContainer && !isOSType(typ) {
			continue
		}

		// Create one slot per (entry name, source) pair — no implicit override
		seen := make(map[string]bool)
		for _, e := range entries {
			if e.Type != typ {
				continue
			}
			key := e.Name + ":" + e.SourceID.String()
			if seen[key] {
				continue
			}
			seen[key] = true

			var slotName, destPath string
			if conv.IsSingleton && conv.Dir == "" {
				slotName = conv.SingletonName
				destPath = conv.SingletonName
			} else if conv.Dir != "" {
				slotName = e.Name
				if isContainer {
					destPath = osDestPath(typ, e.Name)
				} else {
					destPath = conv.Dir + "/" + conv.FileName(e.Name)
				}
			}
			if slotName == "" {
				continue
			}

			slot := registry.Slot{
				ID:         uuid.New(),
				ConsumerID: c.ID,
				Name:       slotName,
				DestPath:   destPath,
			}
			_ = reg.RegisterSlot(slot)

			// Link entries to this slot
			if conv.IsSingleton && conv.Dir == "" {
				// Singleton slot: link all entries of this type from all sources
				for _, e2 := range entries {
					if e2.Type != typ {
						continue
					}
					priority := registry.ScopePriority[sourceScope[e2.SourceID]]
					_ = reg.LinkEntrySlot(e2.ID, slot.ID, priority)
				}
				break // only one singleton slot
			} else {
				// Non-singleton: link only this specific entry
				priority := registry.ScopePriority[sourceScope[e.SourceID]]
				_ = reg.LinkEntrySlot(e.ID, slot.ID, priority)
			}
		}
	}
	return nil
}

func isOSType(typ registry.CapabilityType) bool {
	switch typ {
	case registry.TypeZshConfig, registry.TypeNvimConfig, registry.TypeWeztermConfig, registry.TypeP10kConfig, registry.TypeHelixConfig:
		return true
	}
	return false
}

func osDestPath(typ registry.CapabilityType, name string) string {
	switch typ {
	case registry.TypeZshConfig:
		return "." + name
	case registry.TypeNvimConfig:
		return ".config/nvim/" + name + ".lua"
	case registry.TypeWeztermConfig:
		return ".config/wezterm/" + name + ".lua"
	case registry.TypeP10kConfig:
		return ".p10k.zsh"
	case registry.TypeHelixConfig:
		return ".config/helix/" + name + ".toml"
	}
	return name
}