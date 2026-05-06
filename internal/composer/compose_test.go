package composer_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/composer"
)

func TestOverrideLastEntryWins(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	slots := []composer.Slot{{
		ID:              slotID,
		ConsumerID:      consumerID,
		Name:            "no-push",
		DestinationPath: "rules/no-push.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: uuid.New(), Mode: composer.ModeOverride},
			{ID: uuid.New(), Mode: composer.ModeOverride},
		},
	}

	plan, err := composer.Compose(consumerID, slots, entries)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(plan.Files))
	}
	// Last entry in slot order wins
	if plan.Files[0].SourceEntries[0] != entries[slotID][1].ID {
		t.Error("last entry should win")
	}
	if plan.Files[0].Mode != composer.ModeOverride {
		t.Error("mode should be override")
	}
}

func TestConcatenationAllEntries(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	slots := []composer.Slot{{
		ID:              slotID,
		ConsumerID:      consumerID,
		Name:            "APPEND_SYSTEM",
		DestinationPath: "APPEND_SYSTEM.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: uuid.New(), Mode: composer.ModeConcatenation},
			{ID: uuid.New(), Mode: composer.ModeConcatenation},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	if len(plan.Files[0].SourceEntries) != 2 {
		t.Errorf("got %d source entries, want 2", len(plan.Files[0].SourceEntries))
	}
	if plan.Files[0].Mode != composer.ModeConcatenation {
		t.Error("mode should be concatenation")
	}
}

func TestSlotModeOverridesEntryMode(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()
	concatMode := composer.ModeConcatenation

	slots := []composer.Slot{{
		ID:              slotID,
		ConsumerID:      consumerID,
		Name:            "no-push",
		DestinationPath: "rules/no-push.md",
		ComposeMode:     &concatMode,
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: uuid.New(), Mode: composer.ModeOverride},
			{ID: uuid.New(), Mode: composer.ModeOverride},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	if plan.Files[0].Mode != composer.ModeConcatenation {
		t.Error("slot mode should override entry modes")
	}
	if len(plan.Files[0].SourceEntries) != 2 {
		t.Errorf("got %d source entries, want 2", len(plan.Files[0].SourceEntries))
	}
}

func TestCrossTypeComposition(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	// Rule entries (override) and APPEND_SYSTEM (concat) in the same slot
	// Slot mode unifies to concatenation
	concatMode := composer.ModeConcatenation
	slots := []composer.Slot{{
		ID:              slotID,
		ConsumerID:      consumerID,
		Name:            "output",
		DestinationPath: "output.md",
		ComposeMode:     &concatMode,
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: uuid.New(), Mode: composer.ModeOverride},
			{ID: uuid.New(), Mode: composer.ModeConcatenation},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	if plan.Files[0].Mode != composer.ModeConcatenation {
		t.Error("slot should unify to concatenation")
	}
	if len(plan.Files[0].SourceEntries) != 2 {
		t.Errorf("got %d source entries, want 2", len(plan.Files[0].SourceEntries))
	}
}

func TestEmptySlot(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	slots := []composer.Slot{{
		ID:              slotID,
		ConsumerID:      consumerID,
		Name:            "empty",
		DestinationPath: "empty.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	if len(plan.Files) != 0 {
		t.Errorf("got %d files, want 0", len(plan.Files))
	}
}

func TestComposeIsDeterministic(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	slots := []composer.Slot{{
		ID:              slotID,
		ConsumerID:      consumerID,
		Name:            "rule",
		DestinationPath: "rules/r.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {{ID: uuid.New(), Mode: composer.ModeOverride}},
	}

	p1, _ := composer.Compose(consumerID, slots, entries)
	p2, _ := composer.Compose(consumerID, slots, entries)
	if p1.Files[0].SlotID != p2.Files[0].SlotID {
		t.Error("identical inputs should produce identical outputs")
	}
}
