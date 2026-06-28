package registry_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/registry"
)

// testRegistry is a helper that runs the standard test suite against any Registry implementation.
func testRegistry(t *testing.T, newReg func() (registry.Registry, func())) {
	t.Helper()

	t.Run("RegisterAndResolveConsumer", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		c := registry.Consumer{
			ID:    uuid.New(),
			Alias: "test-repo",
			Path:  "/workspace/test-repo",
			Kind:  registry.ConsumerRepo,
		}
		if err := reg.RegisterConsumer(c); err != nil {
			t.Fatalf("RegisterConsumer: %v", err)
		}

		got, err := reg.ResolveConsumer("test-repo")
		if err != nil {
			t.Fatalf("ResolveConsumer by alias: %v", err)
		}
		if got.ID != c.ID || got.Path != c.Path {
			t.Errorf("got %+v, want %+v", got, c)
		}
	})

	t.Run("ResolveConsumerByPath", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		c := registry.Consumer{
			ID:    uuid.New(),
			Alias: "test-repo",
			Path:  "/workspace/test-repo",
			Kind:  registry.ConsumerRepo,
		}
		reg.RegisterConsumer(c)

		got, err := reg.ResolveConsumer("/workspace/test-repo")
		if err != nil {
			t.Fatalf("ResolveConsumer by path: %v", err)
		}
		if got.ID != c.ID {
			t.Errorf("got ID %v, want %v", got.ID, c.ID)
		}
	})

	t.Run("ResolveConsumerNotFound", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		_, err := reg.ResolveConsumer("nonexistent")
		if err != registry.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("ListConsumers", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		c1 := registry.Consumer{ID: uuid.New(), Alias: "a", Path: "/a", Kind: registry.ConsumerRepo}
		c2 := registry.Consumer{ID: uuid.New(), Alias: "b", Path: "/b", Kind: registry.ConsumerRepo}
		reg.RegisterConsumer(c1)
		reg.RegisterConsumer(c2)

		list, err := reg.ListConsumers()
		if err != nil {
			t.Fatalf("ListConsumers: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("got %d consumers, want 2", len(list))
		}
		if list[0].Alias != "a" || list[1].Alias != "b" {
			t.Errorf("not sorted: %v", list)
		}
	})

	t.Run("RegisterConsumerUpsert", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		id := uuid.New()
		c := registry.Consumer{ID: id, Alias: "repo", Path: "/old", DeployRoot: "/old/.omp", Kind: registry.ConsumerRepo}
		if err := reg.RegisterConsumer(c); err != nil {
			t.Fatalf("RegisterConsumer (insert): %v", err)
		}

		c.Path = "/new"
		c.DeployRoot = "/new/.omp"
		if err := reg.RegisterConsumer(c); err != nil {
			t.Fatalf("RegisterConsumer (upsert): %v", err)
		}

		got, err := reg.ResolveConsumer("repo")
		if err != nil {
			t.Fatalf("ResolveConsumer: %v", err)
		}
		if got.Path != "/new" {
			t.Errorf("Path = %q, want /new", got.Path)
		}
		if got.DeployRoot != "/new/.omp" {
			t.Errorf("DeployRoot = %q, want /new/.omp", got.DeployRoot)
		}
	})

	t.Run("RegisterAndResolveSources", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		s := registry.Source{
			ID:    uuid.New(),
			Alias: "carrel-omp",
			Path:  "/workspace/carrel/omp",
			Scope: registry.ScopeUniversal,
			Kind:  registry.SourceGitBacked,
		}
		if err := reg.RegisterSource(s); err != nil {
			t.Fatalf("RegisterSource: %v", err)
		}

		sources, err := reg.ResolveSources(uuid.New())
		if err != nil {
			t.Fatalf("ResolveSources: %v", err)
		}
		if len(sources) != 1 {
			t.Fatalf("got %d sources, want 1", len(sources))
		}
		if sources[0].Alias != "carrel-omp" {
			t.Errorf("got %s, want carrel-omp", sources[0].Alias)
		}
	})

	t.Run("RegisterSourceUpsert", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		id := uuid.New()
		s := registry.Source{ID: id, Alias: "src", Path: "/old", Scope: registry.ScopeUniversal, Kind: registry.SourceGitBacked}
		if err := reg.RegisterSource(s); err != nil {
			t.Fatalf("RegisterSource (insert): %v", err)
		}

		s.Path = "/new"
		s.Scope = registry.ScopeTargetSpecific
		if err := reg.RegisterSource(s); err != nil {
			t.Fatalf("RegisterSource (upsert): %v", err)
		}

		sources, _ := reg.ListSources()
		if len(sources) != 1 {
			t.Fatalf("got %d sources, want 1", len(sources))
		}
		if sources[0].Path != "/new" {
			t.Errorf("Path = %q, want /new", sources[0].Path)
		}
		if sources[0].Scope != registry.ScopeTargetSpecific {
			t.Errorf("Scope = %d, want ScopeTargetSpecific", sources[0].Scope)
		}
	})

	t.Run("UniversalAndTargetSpecificOrdering", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		c := registry.Consumer{ID: uuid.New(), Alias: "c", Path: "/c"}
		reg.RegisterConsumer(c)

		s1 := registry.Source{ID: uuid.New(), Alias: "universal", Path: "/u", Scope: registry.ScopeUniversal}
		s2 := registry.Source{ID: uuid.New(), Alias: "target", Path: "/t", Scope: registry.ScopeTargetSpecific}
		reg.RegisterSource(s1)
		reg.RegisterSource(s2)

		// Link target-specific source to consumer
		reg.LinkConsumerSource(c.ID, s2.ID)

		sources, err := reg.ResolveSources(c.ID)
		if err != nil {
			t.Fatalf("ResolveSources: %v", err)
		}
		if len(sources) != 2 {
			t.Fatalf("got %d sources, want 2", len(sources))
		}
		if sources[0].Scope != registry.ScopeUniversal {
			t.Error("universal source not first")
		}
	})

	t.Run("LinkConsumerSourceIdempotent", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		c := registry.Consumer{ID: uuid.New(), Alias: "c", Path: "/c"}
		reg.RegisterConsumer(c)
		s := registry.Source{ID: uuid.New(), Alias: "s", Path: "/s", Scope: registry.ScopeTargetSpecific}
		reg.RegisterSource(s)

		if err := reg.LinkConsumerSource(c.ID, s.ID); err != nil {
			t.Fatalf("LinkConsumerSource (first): %v", err)
		}
		if err := reg.LinkConsumerSource(c.ID, s.ID); err != nil {
			t.Fatalf("LinkConsumerSource (second): %v", err)
		}
	})

	t.Run("RegisterAndResolveEntries", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		s := registry.Source{ID: uuid.New(), Alias: "s", Path: "/s"}
		reg.RegisterSource(s)

		e := registry.Entry{
			ID:           uuid.New(),
			SourceID:     s.ID,
			Name:         "no-push-master",
			Type:         registry.TypeRule,
			RelativePath: "rules/no-push-master.md",
			ContentHash:  42,
		}
		if err := reg.RegisterEntry(e); err != nil {
			t.Fatalf("RegisterEntry: %v", err)
		}

		entries, err := reg.ResolveEntries([]uuid.UUID{s.ID})
		if err != nil {
			t.Fatalf("ResolveEntries: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Name != "no-push-master" {
			t.Errorf("got %s, want no-push-master", entries[0].Name)
		}
	})

	t.Run("RegisterEntryUpsert", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		s := registry.Source{ID: uuid.New(), Alias: "s", Path: "/s"}
		reg.RegisterSource(s)

		id := uuid.New()
		e := registry.Entry{
			ID:           id,
			SourceID:     s.ID,
			Name:         "old-name",
			Type:         registry.TypeRule,
			RelativePath: "rules/old-name.md",
			ContentHash:  100,
		}
		if err := reg.RegisterEntry(e); err != nil {
			t.Fatalf("RegisterEntry (insert): %v", err)
		}

		// Re-register same ID with changed name, path, and hash.
		e.Name = "new-name"
		e.RelativePath = "rules/new-name.md"
		e.ContentHash = 999
		if err := reg.RegisterEntry(e); err != nil {
			t.Fatalf("RegisterEntry (upsert): %v", err)
		}

		entries, err := reg.ResolveEntries([]uuid.UUID{s.ID})
		if err != nil {
			t.Fatalf("ResolveEntries: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Name != "new-name" {
			t.Errorf("Name = %q, want new-name", entries[0].Name)
		}
		if entries[0].RelativePath != "rules/new-name.md" {
			t.Errorf("RelativePath = %q, want rules/new-name.md", entries[0].RelativePath)
		}
		if entries[0].ContentHash != 999 {
			t.Errorf("ContentHash = %d, want 999", entries[0].ContentHash)
		}
	})

	t.Run("RemoveEntry", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		s := registry.Source{ID: uuid.New(), Alias: "s", Path: "/s"}
		reg.RegisterSource(s)

		e := registry.Entry{ID: uuid.New(), SourceID: s.ID, Name: "x",
			Type: registry.TypeRule, RelativePath: "r/x.md"}
		reg.RegisterEntry(e)

		if err := reg.RemoveEntry(e.ID); err != nil {
			t.Fatalf("RemoveEntry: %v", err)
		}

		entries, _ := reg.ResolveEntries([]uuid.UUID{s.ID})
		if len(entries) != 0 {
			t.Errorf("got %d entries after remove, want 0", len(entries))
		}
	})

	t.Run("RemoveEntryNotFound", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		err := reg.RemoveEntry(uuid.New())
		if err != registry.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})


	t.Run("DeploymentLifecycle", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		consumerID := uuid.New()
		c := registry.Consumer{ID: consumerID, Alias: "c", Path: "/c"}
		reg.RegisterConsumer(c)

		d := registry.Deployment{
			ID:          uuid.New(),
			ConsumerID:  consumerID,
			AttemptedAt: time.Now(),
			ConfigHash:  12345,
		}

		// Failed deployment (no SucceededAt)
		if err := reg.RecordDeployment(d, nil); err != nil {
			t.Fatalf("RecordDeployment (failed): %v", err)
		}

		_, _, err := reg.LastDeployment(consumerID)
		if err != registry.ErrNoDeployment {
			t.Errorf("got %v, want ErrNoDeployment", err)
		}
	})

	t.Run("SlotPriorityOrdering", func(t *testing.T) {
		reg, cleanup := newReg()
		defer cleanup()

		c := registry.Consumer{ID: uuid.New(), Alias: "c", Path: "/c"}
		reg.RegisterConsumer(c)

		s := registry.Source{ID: uuid.New(), Alias: "s", Path: "/s"}
		reg.RegisterSource(s)

		eLow := registry.Entry{ID: uuid.New(), SourceID: s.ID, Name: "rule", Type: registry.TypeRule, RelativePath: "r.md"}
		eHigh := registry.Entry{ID: uuid.New(), SourceID: s.ID, Name: "rule", Type: registry.TypeRule, RelativePath: "r2.md"}
		reg.RegisterEntry(eLow)
		reg.RegisterEntry(eHigh)

		slot := registry.Slot{ID: uuid.New(), ConsumerID: c.ID, Name: "rule", DestPath: "r.md"}
		reg.RegisterSlot(slot)

		// Link with explicit priorities (low then high)
		reg.LinkEntrySlot(eLow.ID, slot.ID, 0)
		reg.LinkEntrySlot(eHigh.ID, slot.ID, 2)

		entries, err := reg.ResolveEntrySlots(slot.ID)
		if err != nil {
			t.Fatalf("ResolveEntrySlots: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("got %d entries, want 2", len(entries))
		}
		if entries[0].Priority != 0 || entries[0].ID != eLow.ID {
			t.Error("first entry should be priority 0 (eLow)")
		}
		if entries[1].Priority != 2 || entries[1].ID != eHigh.ID {
			t.Error("second entry should be priority 2 (eHigh)")
		}
	})
}

func TestMemRegistry(t *testing.T) {
	testRegistry(t, func() (registry.Registry, func()) {
		return registry.NewMemRegistry(), func() {}
	})
}

func TestDoltRegistry(t *testing.T) {
	testRegistry(t, func() (registry.Registry, func()) {
		tmpDir, err := os.MkdirTemp("", "carrel-test-*")
		if err != nil {
			t.Fatal(err)
		}
		dsn := "file://" + tmpDir + "?commitname=Test&commitemail=test@localhost&database=registry"
		reg, err := registry.NewDoltRegistry(dsn)
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("NewDoltRegistry: %v", err)
		}
		// Create database and apply schema
		if _, err := reg.DB().Exec("CREATE DATABASE registry"); err != nil {
			reg.Close()
			os.RemoveAll(tmpDir)
			t.Fatalf("CREATE DATABASE: %v", err)
		}
		if err := registry.ApplyMigrations(reg); err != nil {
			reg.Close()
			os.RemoveAll(tmpDir)
			t.Fatalf("ApplyMigrations: %v", err)
		}
		return reg, func() {
			reg.Close()
			os.RemoveAll(tmpDir)
		}
	})
}

func TestUnlinkAllEntrySlots(t *testing.T) {
	reg := registry.NewMemRegistry()
	defer reg.Close()

	// Setup: register a source, entry, and slot.
	s := registry.Source{ID: uuid.New(), Alias: "s", Path: "/s"}
	if err := reg.RegisterSource(s); err != nil {
		t.Fatalf("RegisterSource: %v", err)
	}

	e := registry.Entry{
		ID:           uuid.New(),
		SourceID:     s.ID,
		Name:         "rule",
		Type:         registry.TypeRule,
		RelativePath: "r.md",
	}
	if err := reg.RegisterEntry(e); err != nil {
		t.Fatalf("RegisterEntry: %v", err)
	}

	c := registry.Consumer{ID: uuid.New(), Alias: "c", Path: "/c"}
	if err := reg.RegisterConsumer(c); err != nil {
		t.Fatalf("RegisterConsumer: %v", err)
	}

	slot := registry.Slot{ID: uuid.New(), ConsumerID: c.ID, Name: "rule", DestPath: "r.md"}
	if err := reg.RegisterSlot(slot); err != nil {
		t.Fatalf("RegisterSlot: %v", err)
	}

	// Link entry to slot.
	if err := reg.LinkEntrySlot(e.ID, slot.ID, 0); err != nil {
		t.Fatalf("LinkEntrySlot: %v", err)
	}

	// Unlink all entry slots.
	if err := reg.UnlinkAllEntrySlots(e.ID); err != nil {
		t.Fatalf("UnlinkAllEntrySlots: %v", err)
	}

	// Verify zero slot links remain.
	entries, err := reg.ResolveEntrySlots(slot.ID)
	if err != nil {
		t.Fatalf("ResolveEntrySlots: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries after UnlinkAllEntrySlots, want 0", len(entries))
	}

	// UnlinkAllEntrySlots on a nonexistent entry should be a no-op, no error.
	if err := reg.UnlinkAllEntrySlots(uuid.New()); err != nil {
		t.Errorf("UnlinkAllEntrySlots on nonexistent entry: got %v, want nil", err)
	}
}
