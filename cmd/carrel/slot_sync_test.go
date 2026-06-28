package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/authoring"
	"github.com/dichodaemon/carrel/internal/registry"
)

// captureStdout runs fn while capturing os.Stdout. Returns the captured output.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestSlotSyncAllDryRun(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "my-source")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	reg := registry.NewMemRegistry()

	src := registry.Source{
		ID:    uuid.New(),
		Alias: "my-source",
		Path:  srcDir,
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	if err := reg.RegisterSource(src); err != nil {
		t.Fatalf("register source: %v", err)
	}

	consumer := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "test-consumer",
		Path:       filepath.Join(dir, "consumer"),
		DeployRoot: filepath.Join(dir, "consumer", ".omp"),
		Kind:       registry.ConsumerRepo,
	}
	if err := reg.RegisterConsumer(consumer); err != nil {
		t.Fatalf("register consumer: %v", err)
	}

	if err := reg.LinkConsumerSource(consumer.ID, src.ID); err != nil {
		t.Fatalf("link consumer-source: %v", err)
	}

	entry, err := authoring.AddEntry(reg, registry.TypeRule, "my-rule", "my-source", []byte("# My Rule\n"))
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	_ = entry

	output := captureStdout(func() {
		_ = runSlotSyncAll(reg, "test-consumer", true, nil)
	})

	if !strings.Contains(output, "unwired entries") {
		t.Errorf("expected 'unwired entries' in output, got: %s", output)
	}
	if !strings.Contains(output, "my-source:rule:my-rule") {
		t.Errorf("expected 'my-source:rule:my-rule' in output, got: %s", output)
	}

	// Verify no slots were created
	slots, err := reg.ResolveSlots(consumer.ID)
	if err != nil {
		t.Fatalf("ResolveSlots: %v", err)
	}
	if len(slots) != 0 {
		t.Errorf("expected 0 slots, got %d: %+v", len(slots), slots)
	}

	// Verify no entry-slot links exist
	for _, slot := range slots {
		linked, err := reg.ResolveEntrySlots(slot.ID)
		if err != nil {
			t.Fatalf("ResolveEntrySlots: %v", err)
		}
		if len(linked) != 0 {
			t.Errorf("expected 0 linked entries for slot %q, got %d", slot.Name, len(linked))
		}
	}
}

func TestSlotSyncAll(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "my-source")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	reg := registry.NewMemRegistry()

	src := registry.Source{
		ID:    uuid.New(),
		Alias: "my-source",
		Path:  srcDir,
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	if err := reg.RegisterSource(src); err != nil {
		t.Fatalf("register source: %v", err)
	}

	consumer := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "test-consumer",
		Path:       filepath.Join(dir, "consumer"),
		DeployRoot: filepath.Join(dir, "consumer", ".omp"),
		Kind:       registry.ConsumerRepo,
	}
	if err := reg.RegisterConsumer(consumer); err != nil {
		t.Fatalf("register consumer: %v", err)
	}

	if err := reg.LinkConsumerSource(consumer.ID, src.ID); err != nil {
		t.Fatalf("link consumer-source: %v", err)
	}

	entry, err := authoring.AddEntry(reg, registry.TypeRule, "my-rule", "my-source", []byte("# My Rule\n"))
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	output := captureStdout(func() {
		if err := runSlotSyncAll(reg, "test-consumer", false, nil); err != nil {
			t.Fatalf("runSlotSyncAll: %v", err)
		}
	})

	if !strings.Contains(output, "wired 1 entries") {
		t.Errorf("expected 'wired 1 entries' in output, got: %s", output)
	}

	// Verify a slot was auto-created for the entry
	slots, err := reg.ResolveSlots(consumer.ID)
	if err != nil {
		t.Fatalf("ResolveSlots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(slots))
	}
	if slots[0].Name != "my-rule" {
		t.Errorf("expected slot name 'my-rule', got %q", slots[0].Name)
	}

	// Verify entry is linked to the slot
	linked, err := reg.ResolveEntrySlots(slots[0].ID)
	if err != nil {
		t.Fatalf("ResolveEntrySlots: %v", err)
	}
	if len(linked) != 1 {
		t.Fatalf("expected 1 linked entry, got %d", len(linked))
	}
	if linked[0].ID != entry.ID {
		t.Errorf("expected linked entry ID %s, got %s", entry.ID, linked[0].ID)
	}

}

func TestSlotSyncAllExclude(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "my-source")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	reg := registry.NewMemRegistry()

	src := registry.Source{
		ID:    uuid.New(),
		Alias: "my-source",
		Path:  srcDir,
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	if err := reg.RegisterSource(src); err != nil {
		t.Fatalf("register source: %v", err)
	}

	consumer := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "test-consumer",
		Path:       filepath.Join(dir, "consumer"),
		DeployRoot: filepath.Join(dir, "consumer", ".omp"),
		Kind:       registry.ConsumerRepo,
	}
	if err := reg.RegisterConsumer(consumer); err != nil {
		t.Fatalf("register consumer: %v", err)
	}

	if err := reg.LinkConsumerSource(consumer.ID, src.ID); err != nil {
		t.Fatalf("link consumer-source: %v", err)
	}

	keepEntry, err := authoring.AddEntry(reg, registry.TypeRule, "keep-me", "my-source", []byte("# Keep\n"))
	if err != nil {
		t.Fatalf("AddEntry keep-me: %v", err)
	}

	_, err = authoring.AddEntry(reg, registry.TypeRule, "exclude-me", "my-source", []byte("# Exclude\n"))
	if err != nil {
		t.Fatalf("AddEntry exclude-me: %v", err)
	}

	output := captureStdout(func() {
		if err := runSlotSyncAll(reg, "test-consumer", false, []string{"my-source:rule:exclude-me"}); err != nil {
			t.Fatalf("runSlotSyncAll: %v", err)
		}
	})

	if !strings.Contains(output, "wired 1 entries") {
		t.Errorf("expected 'wired 1 entries' in output, got: %s", output)
	}

	// Verify keep-me got wired (slot exists and entry linked)
	slots, err := reg.ResolveSlots(consumer.ID)
	if err != nil {
		t.Fatalf("ResolveSlots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(slots))
	}
	if slots[0].Name != "keep-me" {
		t.Errorf("expected slot name 'keep-me', got %q", slots[0].Name)
	}

	linked, err := reg.ResolveEntrySlots(slots[0].ID)
	if err != nil {
		t.Fatalf("ResolveEntrySlots: %v", err)
	}
	if len(linked) != 1 {
		t.Fatalf("expected 1 linked entry, got %d", len(linked))
	}
	if linked[0].ID != keepEntry.ID {
		t.Errorf("expected linked entry ID %s, got %s", keepEntry.ID, linked[0].ID)
	}

}

func TestRemovedCommands(t *testing.T) {
	rootCmd := &cobra.Command{Use: "carrel"}
	rootCmd.AddCommand(configCmd())

	// "config edit" should be unknown — subcommand removed
	cmd, _, err := rootCmd.Find([]string{"config", "edit"})
	if err == nil && cmd.Name() == "edit" {
		t.Error("'config edit' should be unknown but edit command was found")
	}

	// "config update" should be unknown — subcommand removed
	cmd, _, err = rootCmd.Find([]string{"config", "update"})
	if err == nil && cmd.Name() == "update" {
		t.Error("'config update' should be unknown but update command was found")
	}

	// "verify" should be unknown — command removed
	_, _, err = rootCmd.Find([]string{"verify"})
	if err == nil {
		t.Error("'verify' should be unknown but verify command was found")
	}
}
