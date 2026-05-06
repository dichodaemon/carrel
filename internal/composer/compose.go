package composer

import (
	"sort"

	"github.com/google/uuid"
)

// OutputPlan is the result of composition.
type OutputPlan struct {
	Consumer uuid.UUID
	Files    []OutputFile
}

// OutputFile is a single file to deploy.
type OutputFile struct {
	DestinationPath string
	Content         []byte
	ContentHash     int64
	SourceEntries   []uuid.UUID // Which entries contributed
	SlotID          uuid.UUID
	Mode            ComposeMode // Effective compose mode used
}

// ComposeMode is the composition operator.
type ComposeMode int

const (
	ModeOverride      ComposeMode = 0
	ModeConcatenation ComposeMode = 1
)

// Compose produces an OutputPlan from a consumer's slots and their member entries.
// It is convention-agnostic: no CapabilityType, no Convention lookups, no source scope.
func Compose(consumerID uuid.UUID, slots []Slot, entrySlots map[uuid.UUID][]Entry) (OutputPlan, error) {
	var plan OutputPlan
	plan.Consumer = consumerID

	for _, slot := range slots {
		entries := entrySlots[slot.ID]
		if len(entries) == 0 {
			continue
		}

		// Determine effective mode: slot mode wins if set, otherwise first entry's mode
		mode := entries[0].Mode
		if slot.ComposeMode != nil {
			mode = *slot.ComposeMode
		}

		var file OutputFile
		file.SlotID = slot.ID
		file.DestinationPath = slot.DestinationPath
		file.Mode = mode

		switch mode {
		case ModeOverride:
			file.SourceEntries = composeOverrideSlot(entries)
		case ModeConcatenation:
			file.SourceEntries = composeConcatSlot(entries)
		}

		plan.Files = append(plan.Files, file)
	}

	return plan, nil
}

// composeOverrideSlot sorts entries by priority (low to high), then enforces the
// final flag: the highest-priority entry at or below any final flag wins.
// Entries from higher-priority sources than a final entry are excluded.
func composeOverrideSlot(entries []Entry) []uuid.UUID {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Priority < entries[j].Priority
	})

	// Find the cutoff: highest priority at or below a final entry
	cutoff := len(entries)
	for i, e := range entries {
		if e.Final && i+1 < cutoff {
			cutoff = i + 1
		}
	}

	winner := entries[cutoff-1]
	return []uuid.UUID{winner.ID}
}

// composeConcatSlot sorts entries by priority and returns all contributing entries
// in priority order (low to high). final flag excludes higher-priority entries.
func composeConcatSlot(entries []Entry) []uuid.UUID {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Priority < entries[j].Priority
	})

	// Find the cutoff from final flag
	cutoff := len(entries)
	for i, e := range entries {
		if e.Final && i+1 < cutoff {
			cutoff = i + 1
		}
	}

	var ids []uuid.UUID
	for i := 0; i < cutoff; i++ {
		ids = append(ids, entries[i].ID)
	}
	return ids
}

// Slot is a consumer-specific output target.
type Slot struct {
	ID            uuid.UUID
	ConsumerID    uuid.UUID
	Name          string
	DestinationPath string // Resolved path (consumer.DeployRoot + slot.DestPath)
	ComposeMode   *ComposeMode // nil = use entry modes
}

// Entry is a configuration entry with its compose mode.
type Entry struct {
	ID        uuid.UUID
	SourceID  uuid.UUID
	Mode      ComposeMode
	Final     bool
	Priority  int
}
