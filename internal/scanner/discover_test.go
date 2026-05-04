package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/dichodaemon/carrel/internal/registry"
	"github.com/dichodaemon/carrel/internal/scanner"
)

func TestDiscover(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carrel-scanner-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a git repo
	repoPath := filepath.Join(tmpDir, "test-repo")
	os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)
	os.MkdirAll(filepath.Join(repoPath, ".carrel"), 0755)

	// Create a carula symlink
	ompPath := filepath.Join(repoPath, ".omp")
	os.Symlink("/some/target", ompPath)

	// Create a non-git directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "not-a-repo"), 0755)

	reg := registry.NewMemRegistry()
	repos, err := scanner.Discover(reg, tmpDir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}

	r := repos[0]
	if r.Path != repoPath {
		t.Errorf("path: got %s, want %s", r.Path, repoPath)
	}
	if !r.HasCarrel {
		t.Error("HasCarrel should be true")
	}
	if !r.HasCarula {
		t.Error("HasCarula should be true")
	}
	if r.Registered {
		t.Error("should not be registered")
	}
}

func TestDiscoverRegistered(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carrel-scanner-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")
	os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)

	reg := registry.NewMemRegistry()
	c := registry.Consumer{
		ID:    uuid.New(),
		Alias: "test-repo",
		Path:  repoPath,
		Kind:  registry.ConsumerRepo,
	}
	reg.RegisterConsumer(c)

	repos, _ := scanner.Discover(reg, tmpDir)
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if !repos[0].Registered {
		t.Error("should be registered")
	}
	if repos[0].ConsumerID == nil || *repos[0].ConsumerID != c.ID {
		t.Error("ConsumerID mismatch")
	}
}

func TestMigrate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "carrel-scanner-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")
	os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)

	targetDir := filepath.Join(tmpDir, "target-config")
	os.MkdirAll(targetDir, 0755)
	os.Symlink(targetDir, filepath.Join(repoPath, ".omp"))

	reg := registry.NewMemRegistry()
	results, err := scanner.Migrate(reg, tmpDir)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].Success {
		t.Errorf("migration failed: %v", results[0].Error)
	}

	// Verify consumer was registered
	c, err := reg.ResolveConsumer(repoPath)
	if err != nil {
		t.Fatalf("consumer not registered: %v", err)
	}
	if c.Alias != "test-repo" {
		t.Errorf("alias: got %s, want test-repo", c.Alias)
	}
}
