package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/authoring"
	"github.com/dichodaemon/carrel/internal/query"
	"github.com/dichodaemon/carrel/internal/registry"
)

func TestDeploymentReadsDisk(t *testing.T) {
	reg := registry.NewMemRegistry()

	// Create temp directory for source files
	srcDir := t.TempDir()

	// Register a source backed by the temp directory
	src := registry.Source{
		ID:    uuid.New(),
		Alias: "test-source",
		Path:  srcDir,
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceHostLocal,
	}
	if err := reg.RegisterSource(src); err != nil {
		t.Fatalf("register source: %v", err)
	}

	// Register a consumer
	c := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "test-consumer",
		Path:       "/tmp/test-consumer",
		DeployRoot: "/tmp/test-consumer",
		Kind:       registry.ConsumerContainer,
	}
	if err := reg.RegisterConsumer(c); err != nil {
		t.Fatalf("register consumer: %v", err)
	}

	// Register a slot for the consumer
	slot := registry.Slot{
		ID:         uuid.New(),
		ConsumerID: c.ID,
		Name:       "test-slot",
		DestPath:   "output/test.md",
	}
	if err := reg.RegisterSlot(slot); err != nil {
		t.Fatalf("register slot: %v", err)
	}

	// Add an entry via authoring — writes file on disk AND registers the entry
	contentV1 := []byte("version 1")
	entry, err := authoring.AddEntry(reg, registry.TypeRule, "test-rule", src.Alias, contentV1)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	// Link the entry to the slot
	if err := reg.LinkEntrySlot(entry.ID, slot.ID, 0); err != nil {
		t.Fatalf("LinkEntrySlot: %v", err)
	}

	// Modify the source file on disk to "version 2" WITHOUT re-running carrel config add
	contentV2 := []byte("version 2")
	filePath := filepath.Join(srcDir, entry.RelativePath)
	if err := os.WriteFile(filePath, contentV2, 0644); err != nil {
		t.Fatalf("write version 2 to disk: %v", err)
	}

	// Run plan — should read the on-disk content, not the stale registry hash
	q := query.New(reg)
	results, err := q.Plan(c.Alias)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 plan result, got %d: %+v", len(results), results)
	}

	expectedHash := int64(xxhash.Sum64(contentV2))
	if results[0].PlanHash != expectedHash {
		t.Errorf("PlanHash = %d, want %d (hash of 'version 2')", results[0].PlanHash, expectedHash)
	}

	// Verify PlanHash is NOT the stale registry hash from "version 1"
	staleHash := int64(xxhash.Sum64(contentV1))
	if results[0].PlanHash == staleHash {
		t.Errorf("PlanHash matches stale registry hash (%d); expected on-disk content hash (%d)", staleHash, expectedHash)
	}
}
