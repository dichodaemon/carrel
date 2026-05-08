package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dichodaemon/carrel/internal/registry"
)

// checkSlotsConsistency reads the consumer's .carrel/slots.yml and
// cross-references it against the live registry. Returns warnings for
// any declared entries that have no corresponding slot link.
func checkSlotsConsistency(reg registry.Registry, consumerAlias string) []string {
	c, err := reg.ResolveConsumer(consumerAlias)
	if err != nil {
		return nil
	}

	slotsPath := filepath.Join(c.Path, ".carrel", "slots.yml")
	data, err := os.ReadFile(slotsPath)
	if err != nil {
		return nil // file doesn't exist — nothing to check
	}

	sf, err := registry.ParseSlotsFile(data)
	if err != nil {
		return []string{fmt.Sprintf("slots.yml parse error: %v", err)}
	}

	typeToName := map[registry.CapabilityType]string{
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
	}

	sources, _ := reg.ListSources()
	sourceByID := make(map[string]string) // uuid string → alias
	for _, s := range sources {
		sourceByID[s.ID.String()] = s.Alias
	}

	// Get currently linked entry-slot pairs from the registry
	allSlots, _ := reg.ResolveSlots(c.ID)
	linked := make(map[string]bool) // key: "slotName:entryRef"
	for _, slot := range allSlots {
		entries, _ := reg.ResolveEntrySlots(slot.ID)
		for _, se := range entries {
			srcAlias := sourceByID[se.SourceID.String()]
			typeStr := typeToName[se.Type]
			if srcAlias == "" || typeStr == "" {
				continue
			}
			ref := srcAlias + ":" + typeStr + ":" + se.Name
			key := slot.Name + ":" + ref
			linked[key] = true
		}
	}

	var warnings []string
	for _, sd := range sf.Slots {
		for _, ed := range sd.Entries {
			key := sd.Name + ":" + ed.Ref
			if !linked[key] {
				warnings = append(warnings, fmt.Sprintf(
					"slot %q: entry %q declared in slots.yml but not linked in registry",
					sd.Name, ed.Ref))
			}
		}
	}

	return warnings
}
