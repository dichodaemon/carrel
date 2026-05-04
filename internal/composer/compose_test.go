package composer_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/composer"
	"github.com/dichodaemon/carrel/internal/registry"
)

func TestOverrideDeepestScopeWins(t *testing.T) {
	consumer := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/test"}

	universalSrc := registry.Source{ID: uuid.New(), Alias: "universal", Path: "/u", Scope: registry.ScopeUniversal}
	targetSrc := registry.Source{ID: uuid.New(), Alias: "target", Path: "/t", Scope: registry.ScopeTargetSpecific}
	sources := []registry.Source{universalSrc, targetSrc}

	universalRule := registry.Entry{
		ID: uuid.New(), SourceID: universalSrc.ID, Name: "no-push",
		Type: registry.TypeRule, RelativePath: "rules/no-push.md", ContentHash: 1,
	}
	targetRule := registry.Entry{
		ID: uuid.New(), SourceID: targetSrc.ID, Name: "no-push",
		Type: registry.TypeRule, RelativePath: "rules/no-push.md", ContentHash: 2,
	}
	entries := []registry.Entry{universalRule, targetRule}

	plan, err := composer.Compose(consumer, sources, entries)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if len(plan.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(plan.Files))
	}
	if plan.Files[0].SourceEntries[0] != targetRule.ID {
		t.Error("target rule should win")
	}
	if len(plan.Overrides) != 1 {
		t.Fatalf("got %d overrides, want 1", len(plan.Overrides))
	}
	if plan.Overrides[0].Winner != targetRule.ID {
		t.Error("override winner should be target")
	}
	if plan.Overrides[0].Loser != universalRule.ID {
		t.Error("override loser should be universal")
	}
}

func TestFinalFlagPreventsOverride(t *testing.T) {
	consumer := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/test"}

	universalSrc := registry.Source{ID: uuid.New(), Alias: "universal", Path: "/u", Scope: registry.ScopeUniversal}
	targetSrc := registry.Source{ID: uuid.New(), Alias: "target", Path: "/t", Scope: registry.ScopeTargetSpecific}
	sources := []registry.Source{universalSrc, targetSrc}

	universalRule := registry.Entry{
		ID: uuid.New(), SourceID: universalSrc.ID, Name: "no-push",
		Type: registry.TypeRule, RelativePath: "rules/no-push.md", ContentHash: 1,
		Final: true,
	}
	targetRule := registry.Entry{
		ID: uuid.New(), SourceID: targetSrc.ID, Name: "no-push",
		Type: registry.TypeRule, RelativePath: "rules/no-push.md", ContentHash: 2,
	}
	entries := []registry.Entry{universalRule, targetRule}

	plan, err := composer.Compose(consumer, sources, entries)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if plan.Files[0].SourceEntries[0] != universalRule.ID {
		t.Error("universal rule with Final should win over target")
	}
	if plan.Overrides[0].WasFinal != true {
		t.Error("override should record WasFinal")
	}
}

func TestConcatenationMergesInOrder(t *testing.T) {
	consumer := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/test"}

	universalSrc := registry.Source{ID: uuid.New(), Alias: "universal", Path: "/u", Scope: registry.ScopeUniversal}
	targetSrc := registry.Source{ID: uuid.New(), Alias: "target", Path: "/t", Scope: registry.ScopeTargetSpecific}
	sources := []registry.Source{universalSrc, targetSrc}

	universalAS := registry.Entry{
		ID: uuid.New(), SourceID: universalSrc.ID, Name: "APPEND_SYSTEM.md",
		Type: registry.TypeAppendSystem, RelativePath: "APPEND_SYSTEM.md", ContentHash: 1,
	}
	targetAS := registry.Entry{
		ID: uuid.New(), SourceID: targetSrc.ID, Name: "APPEND_SYSTEM.md",
		Type: registry.TypeAppendSystem, RelativePath: "APPEND_SYSTEM.md", ContentHash: 2,
	}
	entries := []registry.Entry{universalAS, targetAS}

	plan, err := composer.Compose(consumer, sources, entries)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if len(plan.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(plan.Files))
	}
	if plan.Files[0].Primitive != registry.PrimitiveConcatenation {
		t.Error("append system should use concatenation")
	}
	if len(plan.Files[0].SourceEntries) != 2 {
		t.Fatalf("got %d source entries, want 2", len(plan.Files[0].SourceEntries))
	}
	// Universal comes first
	if plan.Files[0].SourceEntries[0] != universalAS.ID {
		t.Error("universal append should come first")
	}
	if plan.Files[0].SourceEntries[1] != targetAS.ID {
		t.Error("target append should come second")
	}
}

func TestComposeIsDeterministic(t *testing.T) {
	consumer := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/test"}
	universalSrc := registry.Source{ID: uuid.New(), Alias: "u", Path: "/u", Scope: registry.ScopeUniversal}
	sources := []registry.Source{universalSrc}
	entries := []registry.Entry{{
		ID: uuid.New(), SourceID: universalSrc.ID, Name: "r",
		Type: registry.TypeRule, RelativePath: "rules/r.md", ContentHash: 7,
	}}

	p1, _ := composer.Compose(consumer, sources, entries)
	p2, _ := composer.Compose(consumer, sources, entries)

	if p1.Files[0].ContentHash != p2.Files[0].ContentHash {
		t.Error("identical inputs should produce identical outputs")
	}
}
