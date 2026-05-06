package composer

import (
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

// composeOverrideSlot: last entry in slot order wins (scan assigns in source scope order).
func composeOverrideSlot(entries []Entry) []uuid.UUID {
	winner := entries[len(entries)-1]
	return []uuid.UUID{winner.ID}
}

// composeConcatSlot: all entries contribute in slot order.
func composeConcatSlot(entries []Entry) []uuid.UUID {
	var ids []uuid.UUID
	for _, e := range entries {
		ids = append(ids, e.ID)
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
}
