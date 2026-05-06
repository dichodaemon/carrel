package composer_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/composer"
)

func TestOverrideFinalFlagTruncatesHigherPriority(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	eLow := uuid.New()
	eFinal := uuid.New() // final at priority 1
	eHigh := uuid.New()  // priority 2, should be excluded

	slots := []composer.Slot{{
		ID:              slotID,
		ConsumerID:      consumerID,
		Name:            "no-push",
		DestinationPath: "rules/no-push.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: eLow, Mode: composer.ModeOverride, Priority: 0},
			{ID: eFinal, Mode: composer.ModeOverride, Priority: 1, Final: true},
			{ID: eHigh, Mode: composer.ModeOverride, Priority: 2},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	if plan.Files[0].SourceEntries[0] != eFinal {
		t.Error("final entry should win, higher priority excluded")
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


func TestOverrideHighestPriorityWins(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	eLow := uuid.New()
	eHigh := uuid.New()

	slots := []composer.Slot{{
		ID: slotID, ConsumerID: consumerID, Name: "rule", DestinationPath: "r.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: eLow, Mode: composer.ModeOverride, Priority: 0},
			{ID: eHigh, Mode: composer.ModeOverride, Priority: 3},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	if plan.Files[0].SourceEntries[0] != eHigh {
		t.Error("highest priority entry should win override")
	}
}

func TestOverrideFinalFlagAtHighestPriorityWins(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	eLow := uuid.New()
	eFinal := uuid.New()

	slots := []composer.Slot{{
		ID: slotID, ConsumerID: consumerID, Name: "rule", DestinationPath: "r.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: eLow, Mode: composer.ModeOverride, Priority: 0},
			{ID: eFinal, Mode: composer.ModeOverride, Priority: 2, Final: true},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	if plan.Files[0].SourceEntries[0] != eFinal {
		t.Error("final at highest priority should win")
	}
}

func TestConcatPriorityOrderPreserved(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	e0 := uuid.New()
	e1 := uuid.New()
	e2 := uuid.New()

	slots := []composer.Slot{{
		ID: slotID, ConsumerID: consumerID, Name: "append", DestinationPath: "a.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: e0, Mode: composer.ModeConcatenation, Priority: 0},
			{ID: e2, Mode: composer.ModeConcatenation, Priority: 2},
			{ID: e1, Mode: composer.ModeConcatenation, Priority: 1},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	ids := plan.Files[0].SourceEntries
	if len(ids) != 3 {
		t.Fatalf("got %d source entries, want 3", len(ids))
	}
	if ids[0] != e0 || ids[1] != e1 || ids[2] != e2 {
		t.Error("concat entries should be sorted by priority, not input order")
	}
}

func TestConcatFinalFlagTruncation(t *testing.T) {
	consumerID := uuid.New()
	slotID := uuid.New()

	e0 := uuid.New()
	e1 := uuid.New()
	e2 := uuid.New()

	slots := []composer.Slot{{
		ID: slotID, ConsumerID: consumerID, Name: "append", DestinationPath: "a.md",
	}}
	entries := map[uuid.UUID][]composer.Entry{
		slotID: {
			{ID: e2, Mode: composer.ModeConcatenation, Priority: 2},
			{ID: e0, Mode: composer.ModeConcatenation, Priority: 0},
			{ID: e1, Mode: composer.ModeConcatenation, Priority: 1, Final: true},
		},
	}

	plan, _ := composer.Compose(consumerID, slots, entries)
	ids := plan.Files[0].SourceEntries
	if len(ids) != 2 {
		t.Fatalf("got %d source entries, want 2 (final excludes higher)", len(ids))
	}
	if ids[0] != e0 || ids[1] != e1 {
		t.Error("should have e0 and e1, excluding higher-priority e2")
	}
}