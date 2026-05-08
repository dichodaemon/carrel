package authoring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/registry"
)

// setup creates an in-memory registry with a single registered source
// backed by a temp directory.
func setup(t *testing.T) (registry.Registry, registry.Source) {
	t.Helper()

	dir := t.TempDir()
	reg := registry.NewMemRegistry()

	source := registry.Source{
		ID:    uuid.New(),
		Alias: "test-source",
		Path:  dir,
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceHostLocal,
	}
	if err := reg.RegisterSource(source); err != nil {
		t.Fatalf("RegisterSource: %v", err)
	}

	return reg, source
}

func TestAddEntry(t *testing.T) {
	reg, source := setup(t)

	content := []byte("# Do not push to master\n")
	entry, err := AddEntry(reg, registry.TypeRule, "no-push-master", "test-source", content)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	// Verify entry fields.
	if entry.SourceID != source.ID {
		t.Errorf("SourceID = %v, want %v", entry.SourceID, source.ID)
	}
	if entry.Name != "no-push-master" {
		t.Errorf("Name = %q, want %q", entry.Name, "no-push-master")
	}
	if entry.Type != registry.TypeRule {
		t.Errorf("Type = %v, want %v", entry.Type, registry.TypeRule)
	}
	wantRelPath := "rules/no-push-master.md"
	if entry.RelativePath != wantRelPath {
		t.Errorf("RelativePath = %q, want %q", entry.RelativePath, wantRelPath)
	}
	wantHash := int64(xxhash.Sum64(content))
	if entry.ContentHash != wantHash {
		t.Errorf("ContentHash = %d, want %d", entry.ContentHash, wantHash)
	}
	if entry.CreatedBy != registry.OriginCarrel {
		t.Errorf("CreatedBy = %v, want %v", entry.CreatedBy, registry.OriginCarrel)
	}

	// Verify file exists on disk with correct content.
	fullPath := filepath.Join(source.Path, wantRelPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", fullPath, err)
	}
	if string(data) != string(content) {
		t.Errorf("file content = %q, want %q", string(data), string(content))
	}

	// Verify entry is in registry.
	entries, err := reg.ResolveEntries([]uuid.UUID{source.ID})
	if err != nil {
		t.Fatalf("ResolveEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].ID != entry.ID {
		t.Errorf("registry entry ID = %v, want %v", entries[0].ID, entry.ID)
	}
}

func TestAddEntrySourceNotFound(t *testing.T) {
	reg, _ := setup(t)

	_, err := AddEntry(reg, registry.TypeRule, "test", "nonexistent", []byte("content"))
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestAddEntryUnknownType(t *testing.T) {
	reg, _ := setup(t)

	_, err := AddEntry(reg, registry.CapabilityType(999), "test", "test-source", []byte("content"))
	if err == nil {
		t.Fatal("expected error for unknown capability type")
	}
}

func TestRemoveEntry(t *testing.T) {
	reg, source := setup(t)

	content := []byte("# A rule\n")
	entry, err := AddEntry(reg, registry.TypeRule, "my-rule", "test-source", content)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	// Verify file exists.
	fullPath := filepath.Join(source.Path, entry.RelativePath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("file should exist: %v", err)
	}

	// Remove.
	if err := RemoveEntry(reg, registry.TypeRule, "my-rule"); err != nil {
		t.Fatalf("RemoveEntry: %v", err)
	}

	// Verify file is gone.
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Errorf("file should not exist after remove: %v", err)
	}

	// Verify registry entry is gone.
	entries, err := reg.ResolveEntries([]uuid.UUID{source.ID})
	if err != nil {
		t.Fatalf("ResolveEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries after remove, want 0", len(entries))
	}
}

func TestRemoveEntryNotFound(t *testing.T) {
	reg, _ := setup(t)

	err := RemoveEntry(reg, registry.TypeRule, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}

func TestEditEntry(t *testing.T) {
	reg, source := setup(t)

	originalContent := []byte("# Original\n")
	entry, err := AddEntry(reg, registry.TypeRule, "test-rule", "test-source", originalContent)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	newContent := []byte("# Modified\n")
	if err := EditEntry(reg, registry.TypeRule, "test-rule", newContent); err != nil {
		t.Fatalf("EditEntry: %v", err)
	}

	// Verify file content on disk.
	fullPath := filepath.Join(source.Path, entry.RelativePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(newContent) {
		t.Errorf("file content = %q, want %q", string(data), string(newContent))
	}

	// Verify ContentHash updated in registry.
	entries, err := reg.ResolveEntries([]uuid.UUID{source.ID})
	if err != nil {
		t.Fatalf("ResolveEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	wantHash := int64(xxhash.Sum64(newContent))
	if entries[0].ContentHash != wantHash {
		t.Errorf("ContentHash = %d, want %d", entries[0].ContentHash, wantHash)
	}
}

func TestUpdateEntryMeta(t *testing.T) {
	reg, source := setup(t)

	content := []byte("# Entry\n")
	_, err := AddEntry(reg, registry.TypeRule, "meta-rule", "test-source", content)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	concat := registry.PrimitiveConcatenation
	concatPtr := &concat
	updates := registry.MetaUpdates{
		ComposeMode: &concatPtr,
	}

	if err := UpdateEntryMeta(reg, registry.TypeRule, "meta-rule", updates); err != nil {
		t.Fatalf("UpdateEntryMeta: %v", err)
	}

	// Verify in registry.
	entries, err := reg.ResolveEntries([]uuid.UUID{source.ID})
	if err != nil {
		t.Fatalf("ResolveEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].ComposeMode != concat {
		t.Errorf("ComposeMode = %v, want %v", entries[0].ComposeMode, concat)
	}
}


func TestUpdateEntryMetaNotFound(t *testing.T) {
	reg, _ := setup(t)

	err := UpdateEntryMeta(reg, registry.TypeRule, "nonexistent", registry.MetaUpdates{})
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}

func TestRenameEntry(t *testing.T) {
	reg, source := setup(t)

	content := []byte("# Skill content\n")
	entry, err := AddEntry(reg, registry.TypeSkill, "old-skill", "test-source", content)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	oldFullPath := filepath.Join(source.Path, entry.RelativePath)

	if err := RenameEntry(reg, registry.TypeSkill, "old-skill", "new-skill"); err != nil {
		t.Fatalf("RenameEntry: %v", err)
	}

	// Verify old file is gone.
	if _, err := os.Stat(oldFullPath); !os.IsNotExist(err) {
		t.Errorf("old file should not exist: %v", err)
	}

	// Verify new file exists with correct content.
	newRelPath := "skills/new-skill/SKILL.md"
	newFullPath := filepath.Join(source.Path, newRelPath)
	data, err := os.ReadFile(newFullPath)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", newFullPath, err)
	}
	if string(data) != string(content) {
		t.Errorf("new file content = %q, want %q", string(data), string(content))
	}

	// Verify registry entry: UUID preserved, name/path updated.
	entries, err := reg.ResolveEntries([]uuid.UUID{source.ID})
	if err != nil {
		t.Fatalf("ResolveEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].ID != entry.ID {
		t.Errorf("UUID changed: got %v, want %v", entries[0].ID, entry.ID)
	}
	if entries[0].Name != "new-skill" {
		t.Errorf("Name = %q, want %q", entries[0].Name, "new-skill")
	}
	if entries[0].RelativePath != newRelPath {
		t.Errorf("RelativePath = %q, want %q", entries[0].RelativePath, newRelPath)
	}
}

func TestRenameEntryNotFound(t *testing.T) {
	reg, _ := setup(t)

	err := RenameEntry(reg, registry.TypeRule, "nonexistent", "new-name")
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}
