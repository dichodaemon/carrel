package registry_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/registry"
)

func TestParseSlotsFile(t *testing.T) {
	yaml := `slots:
  - name: APPEND_SYSTEM.md
    dest: APPEND_SYSTEM.md
    compose: concat
    entries:
      - ref: carrel-omp:append-system:APPEND_SYSTEM.md
        mode: concat
      - ref: carrel-omp:append-system:beads-policy.md
  - name: carrel-configuration
    dest: rules/carrel-configuration.md
    compose: override
    entries:
      - ref: carrel-omp:rule:carrel-configuration
`

	sf, err := registry.ParseSlotsFile([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseSlotsFile: %v", err)
	}
	if len(sf.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(sf.Slots))
	}
	if sf.Slots[0].Name != "APPEND_SYSTEM.md" {
		t.Errorf("slot 0 name = %q", sf.Slots[0].Name)
	}
	if sf.Slots[0].Compose != "concat" {
		t.Errorf("slot 0 compose = %q", sf.Slots[0].Compose)
	}
	if len(sf.Slots[0].Entries) != 2 {
		t.Errorf("slot 0 entries = %d", len(sf.Slots[0].Entries))
	}
}

func TestParseSlotsFileEmpty(t *testing.T) {
	_, err := registry.ParseSlotsFile([]byte("slots: []\n"))
	if err == nil {
		t.Fatal("expected error for empty slots")
	}
}

func TestParseSlotsFileInvalid(t *testing.T) {
	_, err := registry.ParseSlotsFile([]byte("not: yaml: ["))
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

func TestParseEntryRef(t *testing.T) {
	tests := []struct {
		ref       string
		wantSrc   string
		wantType  registry.CapabilityType
		wantName  string
		wantError bool
	}{
		{"carrel-omp:rule:no-push", "carrel-omp", registry.TypeRule, "no-push", false},
		{"carrel-omp:append-system:beads-policy.md", "carrel-omp", registry.TypeAppendSystem, "beads-policy.md", false},
		{"carrel-omp:skill:grill-me", "carrel-omp", registry.TypeSkill, "grill-me", false},
		{"src:command:my-cmd", "src", registry.TypeCommand, "my-cmd", false},
		{"src:extension:my-ext", "src", registry.TypeExtension, "my-ext", false},
		{"src:agent:agent1", "src", registry.TypeAgent, "agent1", false},
		{"src:tool:tool1", "src", registry.TypeTool, "tool1", false},
		{"src:hook:hook1", "src", registry.TypeHook, "hook1", false},
		{"src:prompt:prompt1", "src", registry.TypePrompt, "prompt1", false},
		{"src:instruction:inst1", "src", registry.TypeInstruction, "inst1", false},
		{"src:context-file:ctx", "src", registry.TypeContextFile, "ctx", false},
		{"src:zsh:zshrc", "src", registry.TypeZshConfig, "zshrc", false},
		{"src:nvim:init", "src", registry.TypeNvimConfig, "init", false},
		{"src:wezterm:term", "src", registry.TypeWeztermConfig, "term", false},
		{"src:p10k:p10k", "src", registry.TypeP10kConfig, "p10k", false},
		{"no-colon", "", 0, "", true},
		{"src:badtype:name", "", 0, "", true},
	}
	for _, tt := range tests {
		src, typ, name, err := registry.ParseEntryRef(tt.ref)
		if tt.wantError {
			if err == nil {
				t.Errorf("ParseEntryRef(%q) expected error", tt.ref)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseEntryRef(%q): %v", tt.ref, err)
			continue
		}
		if src != tt.wantSrc {
			t.Errorf("ParseEntryRef(%q) src = %q, want %q", tt.ref, src, tt.wantSrc)
		}
		if typ != tt.wantType {
			t.Errorf("ParseEntryRef(%q) typ = %v, want %v", tt.ref, typ, tt.wantType)
		}
		if name != tt.wantName {
			t.Errorf("ParseEntryRef(%q) name = %q, want %q", tt.ref, name, tt.wantName)
		}
	}
}

func TestTypeName(t *testing.T) {
	tests := []struct {
		name string
		want registry.CapabilityType
		ok   bool
	}{
		{"rule", registry.TypeRule, true},
		{"skill", registry.TypeSkill, true},
		{"command", registry.TypeCommand, true},
		{"extension", registry.TypeExtension, true},
		{"agent", registry.TypeAgent, true},
		{"tool", registry.TypeTool, true},
		{"hook", registry.TypeHook, true},
		{"prompt", registry.TypePrompt, true},
		{"instruction", registry.TypeInstruction, true},
		{"context-file", registry.TypeContextFile, true},
		{"append-system", registry.TypeAppendSystem, true},
		{"zsh", registry.TypeZshConfig, true},
		{"nvim", registry.TypeNvimConfig, true},
		{"wezterm", registry.TypeWeztermConfig, true},
		{"p10k", registry.TypeP10kConfig, true},
		{"unknown", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		got, ok := registry.TypeName(tt.name)
		if ok != tt.ok {
			t.Errorf("TypeName(%q) ok = %v, want %v", tt.name, ok, tt.ok)
		}
		if ok && got != tt.want {
			t.Errorf("TypeName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestComposeModeFromString(t *testing.T) {
	tests := []struct {
		in   string
		want registry.ComposeMode
	}{
		{"concat", registry.ModeConcatenation},
		{"concatenation", registry.ModeConcatenation},
		{"override", registry.ModeOverride},
		{"", registry.ModeConcatenation},
		{"unknown", registry.ModeConcatenation},
	}
	for _, tt := range tests {
		got := registry.ComposeModeFromString(tt.in)
		if got != tt.want {
			t.Errorf("ComposeModeFromString(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseSlotDefaults(t *testing.T) {
	yaml := `contributions:
  - slot: APPEND_SYSTEM.md
    entries:
      - append-system:APPEND_SYSTEM.md
      - append-system:beads-policy.md
  - slot: carrel-configuration
    entries:
      - rule:carrel-configuration
`

	sdf, err := registry.ParseSlotDefaults([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseSlotDefaults: %v", err)
	}
	if len(sdf.Contributions) != 2 {
		t.Fatalf("expected 2 contributions, got %d", len(sdf.Contributions))
	}
	if sdf.Contributions[0].Slot != "APPEND_SYSTEM.md" {
		t.Errorf("contribution 0 slot = %q", sdf.Contributions[0].Slot)
	}
	if len(sdf.Contributions[0].Entries) != 2 {
		t.Errorf("contribution 0 entries = %d", len(sdf.Contributions[0].Entries))
	}
}

func TestParseSlotDefaultsInvalid(t *testing.T) {
	_, err := registry.ParseSlotDefaults([]byte("not: yaml: ["))
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

func TestApplySlotsFile(t *testing.T) {
	reg := registry.NewMem()

	// Register a consumer and source
	src := registry.Source{ID: uuid.New(), Alias: "carrel-omp", Path: "/tmp/src", Scope: registry.ScopeUniversal}
	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterSource(src)
	reg.RegisterConsumer(c)
	reg.LinkConsumerSource(c.ID, src.ID)

	// Register entries
	e1 := registry.Entry{ID: uuid.New(), SourceID: src.ID, Name: "APPEND_SYSTEM.md", Type: registry.TypeAppendSystem}
	e2 := registry.Entry{ID: uuid.New(), SourceID: src.ID, Name: "beads-policy.md", Type: registry.TypeAppendSystem}
	reg.RegisterEntry(e1)
	reg.RegisterEntry(e2)

	// Create temp slots.yml
	dir := t.TempDir()
	slotsPath := filepath.Join(dir, "slots.yml")
	content := `slots:
  - name: APPEND_SYSTEM.md
    dest: APPEND_SYSTEM.md
    compose: concat
    entries:
      - ref: carrel-omp:append-system:APPEND_SYSTEM.md
      - ref: carrel-omp:append-system:beads-policy.md
`
	os.WriteFile(slotsPath, []byte(content), 0644)

	warnings, err := registry.ApplySlotsFile(reg, c, slotsPath)
	if err != nil {
		t.Fatalf("ApplySlotsFile: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// Verify slot was created
	slots, _ := reg.ResolveSlots(c.ID)
	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(slots))
	}
	if slots[0].Name != "APPEND_SYSTEM.md" {
		t.Errorf("slot name = %q", slots[0].Name)
	}

	// Verify entries were linked
	linked, _ := reg.ResolveEntrySlots(slots[0].ID)
	if len(linked) != 2 {
		t.Fatalf("expected 2 linked entries, got %d", len(linked))
	}
}

func TestApplySlotsFileMissingEntry(t *testing.T) {
	reg := registry.NewMem()

	src := registry.Source{ID: uuid.New(), Alias: "carrel-omp", Path: "/tmp/src", Scope: registry.ScopeUniversal}
	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterSource(src)
	reg.RegisterConsumer(c)
	reg.LinkConsumerSource(c.ID, src.ID)

	// Register only one entry — the other ref will fail
	e1 := registry.Entry{ID: uuid.New(), SourceID: src.ID, Name: "APPEND_SYSTEM.md", Type: registry.TypeAppendSystem}
	reg.RegisterEntry(e1)

	dir := t.TempDir()
	slotsPath := filepath.Join(dir, "slots.yml")
	content := `slots:
  - name: APPEND_SYSTEM.md
    dest: APPEND_SYSTEM.md
    compose: concat
    entries:
      - ref: carrel-omp:append-system:APPEND_SYSTEM.md
      - ref: carrel-omp:append-system:missing.md
`
	os.WriteFile(slotsPath, []byte(content), 0644)

	warnings, err := registry.ApplySlotsFile(reg, c, slotsPath)
	if err != nil {
		t.Fatalf("ApplySlotsFile: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "missing.md") {
		t.Errorf("warning does not mention missing entry: %s", warnings[0])
	}
}

func TestApplySlotsFileMissingSource(t *testing.T) {
	reg := registry.NewMem()

	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterConsumer(c)

	dir := t.TempDir()
	slotsPath := filepath.Join(dir, "slots.yml")
	content := `slots:
  - name: APPEND_SYSTEM.md
    dest: APPEND_SYSTEM.md
    compose: concat
    entries:
      - ref: no-such-source:append-system:thing.md
`
	os.WriteFile(slotsPath, []byte(content), 0644)

	warnings, err := registry.ApplySlotsFile(reg, c, slotsPath)
	if err != nil {
		t.Fatalf("ApplySlotsFile: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "not registered") {
		t.Errorf("warning does not mention unregistered source: %s", warnings[0])
	}
}

func TestApplySlotsFileNoFile(t *testing.T) {
	reg := registry.NewMem()
	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterConsumer(c)

	warnings, err := registry.ApplySlotsFile(reg, c, "/no/such/file.yml")
	if err != nil {
		t.Fatalf("ApplySlotsFile: %v", err)
	}
	if warnings != nil {
		t.Errorf("expected nil warnings for missing file, got %v", warnings)
	}
}

func TestApplySlotDefaults(t *testing.T) {
	reg := registry.NewMem()

	src := registry.Source{ID: uuid.New(), Alias: "carrel-omp", Path: "/tmp/src", Scope: registry.ScopeUniversal}
	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterSource(src)
	reg.RegisterConsumer(c)

	e1 := registry.Entry{ID: uuid.New(), SourceID: src.ID, Name: "APPEND_SYSTEM.md", Type: registry.TypeAppendSystem}
	e2 := registry.Entry{ID: uuid.New(), SourceID: src.ID, Name: "beads-policy.md", Type: registry.TypeAppendSystem}
	reg.RegisterEntry(e1)
	reg.RegisterEntry(e2)

	dir := t.TempDir()
	defaultsPath := filepath.Join(dir, "slot-defaults.yml")
	content := `contributions:
  - slot: APPEND_SYSTEM.md
    entries:
      - append-system:APPEND_SYSTEM.md
      - append-system:beads-policy.md
`
	os.WriteFile(defaultsPath, []byte(content), 0644)

	warnings, err := registry.ApplySlotDefaults(reg, c, src, defaultsPath)
	if err != nil {
		t.Fatalf("ApplySlotDefaults: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	slots, _ := reg.ResolveSlots(c.ID)
	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(slots))
	}
	linked, _ := reg.ResolveEntrySlots(slots[0].ID)
	if len(linked) != 2 {
		t.Fatalf("expected 2 linked entries, got %d", len(linked))
	}
}

func TestApplySlotDefaultsNoFile(t *testing.T) {
	reg := registry.NewMem()
	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterConsumer(c)
	src := registry.Source{ID: uuid.New(), Alias: "s", Path: "/tmp/s", Scope: registry.ScopeUniversal}

	warnings, err := registry.ApplySlotDefaults(reg, c, src, "/no/such/file.yml")
	if err != nil {
		t.Fatalf("ApplySlotDefaults: %v", err)
	}
	if warnings != nil {
		t.Errorf("expected nil warnings for missing file, got %v", warnings)
	}
}

func TestGenerateSlotsFile(t *testing.T) {
	reg := registry.NewMem()

	src := registry.Source{ID: uuid.New(), Alias: "carrel-omp", Path: "/tmp/src", Scope: registry.ScopeUniversal}
	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterSource(src)
	reg.RegisterConsumer(c)

	e := registry.Entry{ID: uuid.New(), SourceID: src.ID, Name: "APPEND_SYSTEM.md", Type: registry.TypeAppendSystem}
	reg.RegisterEntry(e)

	slot := registry.Slot{ID: uuid.New(), ConsumerID: c.ID, Name: "APPEND_SYSTEM.md", DestPath: "APPEND_SYSTEM.md"}
	reg.RegisterSlot(slot)
	reg.LinkEntrySlot(e.ID, slot.ID, 0)

	sf, err := registry.GenerateSlotsFile(reg, "test")
	if err != nil {
		t.Fatalf("GenerateSlotsFile: %v", err)
	}
	if len(sf.Slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(sf.Slots))
	}
	if sf.Slots[0].Name != "APPEND_SYSTEM.md" {
		t.Errorf("slot name = %q", sf.Slots[0].Name)
	}
	if len(sf.Slots[0].Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(sf.Slots[0].Entries))
	}
	if sf.Slots[0].Compose == "" {
		t.Errorf("compose mode should not be empty (falls back to first entry's mode)")
	}
}

func TestApplySlotsFileIdempotent(t *testing.T) {
	reg := registry.NewMem()

	src := registry.Source{ID: uuid.New(), Alias: "carrel-omp", Path: "/tmp/src", Scope: registry.ScopeUniversal}
	c := registry.Consumer{ID: uuid.New(), Alias: "test", Path: "/tmp/test", DeployRoot: "/tmp/test/.omp", Kind: registry.ConsumerRepo}
	reg.RegisterSource(src)
	reg.RegisterConsumer(c)
	reg.LinkConsumerSource(c.ID, src.ID)

	e := registry.Entry{ID: uuid.New(), SourceID: src.ID, Name: "APPEND_SYSTEM.md", Type: registry.TypeAppendSystem}
	reg.RegisterEntry(e)

	dir := t.TempDir()
	slotsPath := filepath.Join(dir, "slots.yml")
	content := `slots:
  - name: APPEND_SYSTEM.md
    dest: APPEND_SYSTEM.md
    compose: concat
    entries:
      - ref: carrel-omp:append-system:APPEND_SYSTEM.md
`
	os.WriteFile(slotsPath, []byte(content), 0644)

	// First application
	warnings, err := registry.ApplySlotsFile(reg, c, slotsPath)
	if err != nil {
		t.Fatalf("first ApplySlotsFile: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("first pass warnings: %v", warnings)
	}

	// Second application (idempotent)
	warnings, err = registry.ApplySlotsFile(reg, c, slotsPath)
	if err != nil {
		t.Fatalf("second ApplySlotsFile: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("second pass warnings: %v", warnings)
	}

	// Verify only one slot
	slots, _ := reg.ResolveSlots(c.ID)
	if len(slots) != 1 {
		t.Errorf("expected 1 slot, got %d", len(slots))
	}
	// Verify only one link
	linked, _ := reg.ResolveEntrySlots(slots[0].ID)
	if len(linked) != 1 {
		t.Errorf("expected 1 linked entry after idempotent apply, got %d", len(linked))
	}
}
