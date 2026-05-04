package deployer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/composer"
	"github.com/dichodaemon/carrel/internal/deployer"
	"github.com/dichodaemon/carrel/internal/registry"
)

func makePlan(consumerID uuid.UUID, files ...composer.OutputFile) composer.OutputPlan {
	return composer.OutputPlan{Consumer: consumerID, Files: files}
}

func makeFile(dest string, content []byte, sourceID uuid.UUID) composer.OutputFile {
	return composer.OutputFile{
		DestinationPath: dest,
		Content:         content,
		ContentHash:     xxhash.Sum64(content),
		SourceEntries:   []uuid.UUID{sourceID},
		Primitive:       registry.PrimitiveOverride,
	}
}

func writeTemp(t *testing.T, dir, relPath string, content []byte) string {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, content, 0644); err != nil {
		t.Fatal(err)
	}
	return full
}

func mustHash(data []byte) uint64 {
	return xxhash.Sum64(data)
}

func TestDeployWritesFilesNoConflicts(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	file := makeFile(filepath.Join(tmp, "rules/no-push.md"), []byte("# no push"), sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, nil, deployer.ConflictError, false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("unexpected collisions: %+v", results)
	}

	got, err := os.ReadFile(file.DestinationPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "# no push" {
		t.Errorf("got %q, want %q", string(got), "# no push")
	}
}

func TestConflictErrorReturnsWithoutWriting(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	// Pre-write a foreign file
	existing := writeTemp(t, tmp, "rules/no-push.md", []byte("foreign content"))
	newContent := []byte("# carrel content")
	file := makeFile(existing, newContent, sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, nil, deployer.ConflictError, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(results))
	}
	if results[0].Action != deployer.ActionErrored {
		t.Errorf("expected ActionErrored, got %v", results[0].Action)
	}

	// File on disk must be unchanged
	got, _ := os.ReadFile(existing)
	if string(got) != "foreign content" {
		t.Errorf("foreign file was modified: got %q", string(got))
	}
}

func TestConflictBackupCreatesBackup(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	existing := writeTemp(t, tmp, "rules/no-push.md", []byte("foreign content"))
	existingHash := mustHash([]byte("foreign content"))

	file := makeFile(existing, []byte("# carrel content"), sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, nil, deployer.ConflictBackup, false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(results))
	}
	if results[0].Action != deployer.ActionBackedUp {
		t.Errorf("expected ActionBackedUp, got %v", results[0].Action)
	}
	if results[0].ForeignHash != existingHash {
		t.Errorf("ForeignHash %d != %d", results[0].ForeignHash, existingHash)
	}

	// New file written
	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "# carrel content" {
		t.Errorf("got %q, want %q", string(got), "# carrel content")
	}

	// Backup exists
	backup, err := os.ReadFile(existing + ".bak")
	if err != nil {
		t.Fatalf("ReadFile .bak: %v", err)
	}
	if string(backup) != "foreign content" {
		t.Errorf("backup got %q, want %q", string(backup), "foreign content")
	}
}

func TestConflictSkipSkipsPath(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	existing := writeTemp(t, tmp, "rules/no-push.md", []byte("foreign content"))
	skipped := makeFile(existing, []byte("# carrel content"), sourceID)

	otherPath := filepath.Join(tmp, "rules/other.md")
	other := makeFile(otherPath, []byte("# other"), sourceID)

	plan := makePlan(consumerID, skipped, other)

	results, err := deployer.Deploy(plan, nil, deployer.ConflictSkip, false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(results))
	}
	if results[0].Action != deployer.ActionSkipped {
		t.Errorf("expected ActionSkipped, got %v", results[0].Action)
	}

	// Skipped file unchanged
	got, _ := os.ReadFile(existing)
	if string(got) != "foreign content" {
		t.Errorf("skipped file modified: got %q", string(got))
	}

	// Other file written
	got2, _ := os.ReadFile(otherPath)
	if string(got2) != "# other" {
		t.Errorf("other file: got %q, want %q", string(got2), "# other")
	}
}

func TestDryRunNoWrites(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	file := makeFile(filepath.Join(tmp, "rules/no-push.md"), []byte("# carrel content"), sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, nil, deployer.ConflictError, true)
	if err != nil {
		t.Fatalf("Deploy dryRun: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("unexpected collisions: %+v", results)
	}

	if _, err := os.Stat(file.DestinationPath); !os.IsNotExist(err) {
		t.Error("dryRun wrote file to disk")
	}
}

func TestDryRunForeignFileErr(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	existing := writeTemp(t, tmp, "rules/no-push.md", []byte("foreign content"))
	existingHash := mustHash([]byte("foreign content"))

	file := makeFile(existing, []byte("# carrel content"), sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, nil, deployer.ConflictError, true)
	if err != nil {
		t.Fatalf("Deploy dryRun: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(results))
	}
	if results[0].Action != deployer.ActionErrored {
		t.Errorf("expected ActionErrored, got %v", results[0].Action)
	}
	if results[0].ForeignHash != existingHash {
		t.Errorf("ForeignHash %d != %d", results[0].ForeignHash, existingHash)
	}

	// File still intact
	got, _ := os.ReadFile(existing)
	if string(got) != "foreign content" {
		t.Errorf("foreign file modified: got %q", string(got))
	}
}

func TestStaleFilesRemovedWhenHashMatches(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	deploymentID := uuid.New()

	// Write a file that is carrel-owned (in previousClaim)
	stalePath := writeTemp(t, tmp, "rules/old-rule.md", []byte("old content"))
	staleHash := mustHash([]byte("old content"))

	previousClaim := []registry.DeploymentEntry{
		{
			DeploymentID: deploymentID,
			Path:         stalePath,
			ContentHash:  staleHash,
			SourceEntry:  sourceID,
		},
	}

	// Plan with a different file (old-rule is absent)
	file := makeFile(filepath.Join(tmp, "rules/new-rule.md"), []byte("new content"), sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, previousClaim, deployer.ConflictError, false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("unexpected collisions: %+v", results)
	}

	// Old file removed
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Error("stale file not removed")
	}
}

func TestStaleFileHashMismatchLeftUntouched(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()
	deploymentID := uuid.New()

	// Write a file that simulates a previous deployment
	stalePath := writeTemp(t, tmp, "rules/old-rule.md", []byte("original content"))

	// But claim a different hash — simulates external modification
	previousClaim := []registry.DeploymentEntry{
		{
			DeploymentID: deploymentID,
			Path:         stalePath,
			ContentHash:  99999, // wrong hash
			SourceEntry:  sourceID,
		},
	}

	file := makeFile(filepath.Join(tmp, "rules/new-rule.md"), []byte("new content"), sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, previousClaim, deployer.ConflictError, false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 collision from stale mismatch, got %d", len(results))
	}
	if results[0].Action != deployer.ActionSkipped {
		t.Errorf("expected stale file ActionSkipped, got %v", results[0].Action)
	}

	// Stale file still exists
	if _, err := os.Stat(stalePath); os.IsNotExist(err) {
		t.Error("stale file with mismatch hash was removed")
	}
}

func TestStaleFileDryRunDoesNotRemove(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()
	deploymentID := uuid.New()

	stalePath := writeTemp(t, tmp, "rules/old-rule.md", []byte("old content"))
	staleHash := mustHash([]byte("old content"))

	previousClaim := []registry.DeploymentEntry{
		{
			DeploymentID: deploymentID,
			Path:         stalePath,
			ContentHash:  staleHash,
			SourceEntry:  sourceID,
		},
	}

	file := makeFile(filepath.Join(tmp, "rules/new-rule.md"), []byte("new content"), sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, previousClaim, deployer.ConflictError, true)
	if err != nil {
		t.Fatalf("Deploy dryRun: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("unexpected collisions: %+v", results)
	}

	// Stale file must still exist
	if _, err := os.Stat(stalePath); os.IsNotExist(err) {
		t.Error("dryRun removed stale file")
	}
}

func TestCarrelOwnedFileReplaced(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()
	deploymentID := uuid.New()

	oldContent := []byte("carrel v1")
	filePath := writeTemp(t, tmp, "rules/rule.md", oldContent)
	oldHash := mustHash(oldContent)

	previousClaim := []registry.DeploymentEntry{
		{
			DeploymentID: deploymentID,
			Path:         filePath,
			ContentHash:  oldHash,
			SourceEntry:  sourceID,
		},
	}

	newContent := []byte("carrel v2")
	file := makeFile(filePath, newContent, sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, previousClaim, deployer.ConflictError, false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no collisions for carrel-owned file, got: %+v", results)
	}

	got, _ := os.ReadFile(filePath)
	if string(got) != "carrel v2" {
		t.Errorf("got %q, want %q", string(got), "carrel v2")
	}
}

func TestCarrelOwnedButTamperedFileIsForeign(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()
	deploymentID := uuid.New()

	// Simulate a carrel-owned file that was externally modified
	tamperedContent := []byte("someone changed me")
	filePath := writeTemp(t, tmp, "rules/rule.md", tamperedContent)
	tamperedHash := mustHash(tamperedContent)

	previousClaim := []registry.DeploymentEntry{
		{
			DeploymentID: deploymentID,
			Path:         filePath,
			ContentHash:  12345, // claimed hash no longer matches
			SourceEntry:  sourceID,
		},
	}

	newContent := []byte("carrel v2")
	file := makeFile(filePath, newContent, sourceID)
	plan := makePlan(consumerID, file)

	results, err := deployer.Deploy(plan, previousClaim, deployer.ConflictError, false)
	if err == nil {
		t.Fatal("expected error for tampered file with ConflictError policy")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(results))
	}
	if results[0].ForeignHash != tamperedHash {
		t.Errorf("ForeignHash %d != %d", results[0].ForeignHash, tamperedHash)
	}
}

func TestDirectoryAtDestinationErrors(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	dirPath := filepath.Join(tmp, "rules")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	file := makeFile(dirPath, []byte("content"), sourceID)
	plan := makePlan(consumerID, file)

	_, err := deployer.Deploy(plan, nil, deployer.ConflictError, false)
	if err == nil {
		t.Fatal("expected error for directory at destination path")
	}
}

func TestMultipleFilesPartialCollisions(t *testing.T) {
	tmp := t.TempDir()
	consumerID := uuid.New()
	sourceID := uuid.New()

	// Foreign file
	foreignPath := writeTemp(t, tmp, "rules/foreign.md", []byte("foreign"))
	foreignFile := makeFile(foreignPath, []byte("# new foreign"), sourceID)

	// New file
	cleanPath := filepath.Join(tmp, "rules/clean.md")
	cleanFile := makeFile(cleanPath, []byte("# clean"), sourceID)

	plan := makePlan(consumerID, foreignFile, cleanFile)

	results, err := deployer.Deploy(plan, nil, deployer.ConflictSkip, false)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(results))
	}

	// Foreign unchanged
	got, _ := os.ReadFile(foreignPath)
	if string(got) != "foreign" {
		t.Errorf("foreign modified: got %q", string(got))
	}

	// Clean written
	got2, _ := os.ReadFile(cleanPath)
	if string(got2) != "# clean" {
		t.Errorf("clean: got %q", string(got2))
	}
}
