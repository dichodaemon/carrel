package registry

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)
// SlotsFile is the top-level structure of a .carrel/slots.yml file.
type SlotsFile struct {
	Slots []SlotDecl `yaml:"slots"`
}

// SlotDecl declares a slot and its entry links in the file.
type SlotDecl struct {
	Name    string          `yaml:"name"`
	Dest    string          `yaml:"dest"`
	Compose string          `yaml:"compose"`
	Entries []SlotEntryDecl `yaml:"entries"`
}

// SlotEntryDecl is a single entry reference in a slot declaration.
// The Ref field is a compound ID: source-alias:type:name.
// Mode is the per-entry compose mode; empty defaults to the slot's compose.
type SlotEntryDecl struct {
	Ref  string `yaml:"ref"`
	Mode string `yaml:"mode,omitempty"`
}

// ParseSlotsFile parses a YAML byte slice into a SlotsFile.
func ParseSlotsFile(data []byte) (*SlotsFile, error) {
	var sf SlotsFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse slots.yml: %w", err)
	}
	if len(sf.Slots) == 0 {
		return nil, fmt.Errorf("parse slots.yml: no slots declared")
	}
	return &sf, nil
}


// TypeName maps a capability type name string to its CapabilityType constant.
func TypeName(s string) (CapabilityType, bool) {
	switch s {
	case "rule":
		return TypeRule, true
	case "skill":
		return TypeSkill, true
	case "command":
		return TypeCommand, true
	case "extension":
		return TypeExtension, true
	case "agent":
		return TypeAgent, true
	case "tool":
		return TypeTool, true
	case "hook":
		return TypeHook, true
	case "prompt":
		return TypePrompt, true
	case "instruction":
		return TypeInstruction, true
	case "context-file":
		return TypeContextFile, true
	case "append-system":
		return TypeAppendSystem, true
	case "zsh":
		return TypeZshConfig, true
	case "nvim":
		return TypeNvimConfig, true
	case "wezterm":
		return TypeWeztermConfig, true
	case "p10k":
		return TypeP10kConfig, true
	default:
		return 0, false
	}
}
// ParseEntryRef splits a compound ID "source-alias:type:name" into its components.
// The type string is resolved to a CapabilityType. Returns zero values on parse failure.
func ParseEntryRef(ref string) (sourceAlias string, typ CapabilityType, name string, err error) {
	parts := strings.SplitN(ref, ":", 3)
	if len(parts) != 3 {
		return "", 0, "", fmt.Errorf("invalid entry ref %q: expected source:type:name", ref)
	}
	sourceAlias = parts[0]
	typeStr := parts[1]
	name = parts[2]

	typ, ok := TypeName(typeStr)
	if !ok {
		return "", 0, "", fmt.Errorf("unknown type %q in entry ref %q", typeStr, ref)
	}
	return sourceAlias, typ, name, nil
}

// ComposeModeFromString converts a string to ComposeMode. Defaults to ModeConcatenation.
func ComposeModeFromString(s string) ComposeMode {
	switch strings.ToLower(s) {
	case "override":
		return ModeOverride
	case "concat", "concatenation":
		return ModeConcatenation
	default:
		if s != "" {
			// Unknown mode string — log or ignore; default to concat.
		}
		return ModeConcatenation
	}
}

// GenerateSlotsFile reads the consumer's registry state and produces a SlotsFile
// representing the current slot definitions and entry-to-slot links.
func GenerateSlotsFile(reg Registry, consumerAlias string) (*SlotsFile, error) {
	c, err := reg.ResolveConsumer(consumerAlias)
	if err != nil {
		return nil, fmt.Errorf("resolve consumer %q: %w", consumerAlias, err)
	}

	slots, err := reg.ResolveSlots(c.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve slots: %w", err)
	}

	sources, err := reg.ListSources()
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	sourceByID := make(map[uuid.UUID]Source)
	for _, s := range sources {
		sourceByID[s.ID] = s
	}

	// Build type-name-to-string map for serialization
	typeNames := map[CapabilityType]string{
		TypeRule:          "rule",
		TypeSkill:         "skill",
		TypeCommand:       "command",
		TypeExtension:     "extension",
		TypeAgent:          "agent",
		TypeTool:           "tool",
		TypeHook:           "hook",
		TypePrompt:         "prompt",
		TypeInstruction:    "instruction",
		TypeContextFile:    "context-file",
		TypeAppendSystem:   "append-system",
		TypeZshConfig:      "zsh",
		TypeNvimConfig:     "nvim",
		TypeWeztermConfig:  "wezterm",
		TypeP10kConfig:     "p10k",
	}
	composeModeNames := map[ComposeMode]string{
		ModeOverride:      "override",
		ModeConcatenation: "concat",
	}

	var decls []SlotDecl
	for _, slot := range slots {
		entries, eErr := reg.ResolveEntrySlots(slot.ID)
		if eErr != nil {
			continue
		}

		modeStr := ""
		if slot.ComposeMode != nil {
			modeStr = composeModeNames[*slot.ComposeMode]
		} else if len(entries) > 0 {
			modeStr = composeModeNames[entries[0].ComposeMode]
		}

		var entryDecls []SlotEntryDecl
		for _, se := range entries {
			src, ok := sourceByID[se.SourceID]
			if !ok {
				continue
			}
			typeStr, ok := typeNames[se.Type]
			if !ok {
				continue
			}
			ref := src.Alias + ":" + typeStr + ":" + se.Name

			ed := SlotEntryDecl{Ref: ref}
			entryDecls = append(entryDecls, ed)
		}

		if len(entryDecls) == 0 {
			continue
		}

		decls = append(decls, SlotDecl{
			Name:    slot.Name,
			Dest:    slot.DestPath,
			Compose: modeStr,
			Entries: entryDecls,
		})
	}

	return &SlotsFile{Slots: decls}, nil
}

// ApplySlotsFile reads a slots.yml file and applies its declarations to a consumer's
// registry state: creates missing slots, links entries by compound ID, skips idempotent.
// Returns warnings for entries that could not be resolved (source or entry not found).
func ApplySlotsFile(reg Registry, c Consumer, path string) (warnings []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read slots.yml: %w", err)
	}

	sf, err := ParseSlotsFile(data)
	if err != nil {
		return nil, err
	}

	sources, _ := reg.ListSources()
	sourceByAlias := make(map[string]Source)
	for _, s := range sources {
		sourceByAlias[s.Alias] = s
	}

	existingSlots, _ := reg.ResolveSlots(c.ID)
	slotByName := make(map[string]Slot)
	for _, s := range existingSlots {
		slotByName[s.Name] = s
	}

	for _, sd := range sf.Slots {
		slot, slotExists := slotByName[sd.Name]
		if !slotExists {
			slot = Slot{
				ID:         uuid.New(),
				ConsumerID: c.ID,
				Name:       sd.Name,
				DestPath:   sd.Dest,
			}
			if sd.Compose != "" {
				mode := ComposeModeFromString(sd.Compose)
				slot.ComposeMode = &mode
			}
			if err := reg.RegisterSlot(slot); err != nil {
				warnings = append(warnings, fmt.Sprintf("slot %q: register: %v", sd.Name, err))
				continue
			}
		} else if sd.Compose != "" {
			mode := ComposeModeFromString(sd.Compose)
			modePtr := &mode
			_ = reg.UpdateSlot(slot.ID, SlotUpdates{ComposeMode: &modePtr})
		}

		for _, ed := range sd.Entries {
			srcAlias, typ, name, refErr := ParseEntryRef(ed.Ref)
			if refErr != nil {
				warnings = append(warnings, fmt.Sprintf("slot %q: %v", sd.Name, refErr))
				continue
			}
			src, srcOK := sourceByAlias[srcAlias]
			if !srcOK {
				warnings = append(warnings, fmt.Sprintf("slot %q: entry %q: source %q not registered", sd.Name, ed.Ref, srcAlias))
				continue
			}

			entries, _ := reg.ResolveEntries([]uuid.UUID{src.ID})
			var entryID uuid.UUID
			for _, e := range entries {
				if e.Type == typ && e.Name == name {
					entryID = e.ID
					break
				}
			}
			if entryID == uuid.Nil {
				warnings = append(warnings, fmt.Sprintf("slot %q: entry %q not found", sd.Name, ed.Ref))
				continue
			}

			priority := ScopePriority[src.Scope]
			_ = reg.LinkEntrySlot(entryID, slot.ID, priority)
		}
	}

	return warnings, nil
}

// SlotDefaultsFile is the structure of a source's slot-defaults.yml file.
// It declares which entries from this source contribute to which slots.
type SlotDefaultsFile struct {
	Contributions []SlotDefault `yaml:"contributions"`
}

// SlotDefault declares a slot and the entries from this source that feed it.
// Entry references are type:name (source alias is implicit).
type SlotDefault struct {
	Slot    string   `yaml:"slot"`
	Entries []string `yaml:"entries"`
}

// ParseSlotDefaults parses a YAML byte slice into a SlotDefaultsFile.
func ParseSlotDefaults(data []byte) (*SlotDefaultsFile, error) {
	var sf SlotDefaultsFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse slot-defaults.yml: %w", err)
	}
	return &sf, nil
}

// ApplySlotDefaults reads slot-defaults.yml from a source directory and applies
// its contribution declarations to a consumer. Entries are resolved within the
// given source only. Returns warnings for unresolvable entries.
func ApplySlotDefaults(reg Registry, c Consumer, src Source, path string) (warnings []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read slot-defaults.yml: %w", err)
	}

	sdf, err := ParseSlotDefaults(data)
	if err != nil {
		return nil, err
	}

	existingSlots, _ := reg.ResolveSlots(c.ID)
	slotByName := make(map[string]Slot)
	for _, s := range existingSlots {
		slotByName[s.Name] = s
	}

	// Get all entries from this source
	entries, _ := reg.ResolveEntries([]uuid.UUID{src.ID})

	for _, sd := range sdf.Contributions {
		slot, slotExists := slotByName[sd.Slot]
		if !slotExists {
			// Create slot with conventional destination
			slot = Slot{
				ID:         uuid.New(),
				ConsumerID: c.ID,
				Name:       sd.Slot,
				DestPath:   sd.Slot, // default: slot name = dest path; conventions may override
			}
			if err := reg.RegisterSlot(slot); err != nil {
				warnings = append(warnings, fmt.Sprintf("slot %q: register: %v", sd.Slot, err))
				continue
			}
		}

		for _, entryRef := range sd.Entries {
			typeName, name := splitTypeName(entryRef)
			if typeName == "" || name == "" {
				warnings = append(warnings, fmt.Sprintf("slot %q: invalid entry ref %q", sd.Slot, entryRef))
				continue
			}

			var entryID uuid.UUID
			for _, e := range entries {
				if e.Name == name {
					if typStr := typeToName[e.Type]; typStr == typeName {
						entryID = e.ID
						break
					}
				}
			}
			if entryID == uuid.Nil {
				warnings = append(warnings, fmt.Sprintf("slot %q: entry %q not found in source %q", sd.Slot, entryRef, src.Alias))
				continue
			}

			priority := ScopePriority[src.Scope]
			_ = reg.LinkEntrySlot(entryID, slot.ID, priority)
		}
	}

	return warnings, nil
}

// splitTypeName splits "type:name" into its two components.
func splitTypeName(ref string) (typ, name string) {
	idx := strings.IndexByte(ref, ':')
	if idx < 0 {
		return "", ""
	}
	return ref[:idx], ref[idx+1:]
}

// typeToName maps CapabilityType to its string form.
var typeToName = map[CapabilityType]string{
	TypeRule:          "rule",
	TypeSkill:         "skill",
	TypeCommand:       "command",
	TypeExtension:     "extension",
	TypeAgent:         "agent",
	TypeTool:          "tool",
	TypeHook:          "hook",
	TypePrompt:        "prompt",
	TypeInstruction:   "instruction",
	TypeContextFile:   "context-file",
	TypeAppendSystem:  "append-system",
	TypeZshConfig:     "zsh",
	TypeNvimConfig:    "nvim",
	TypeWeztermConfig: "wezterm",
	TypeP10kConfig:    "p10k",
}
