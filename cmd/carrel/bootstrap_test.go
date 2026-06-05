package main

import (
	"testing"

	"github.com/dichodaemon/carrel/internal/registry"
	"github.com/google/uuid"
)

func TestGenerateDefaultSlots_ContainerGetsOSTypes(t *testing.T) {
	reg := registry.NewMemRegistry()

	src := registry.Source{
		ID:    uuid.New(),
		Alias: "carrel-omp",
		Path:  "/workspace/carrel/omp",
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	if err := reg.RegisterSource(src); err != nil {
		t.Fatalf("register source: %v", err)
	}

	c := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "container",
		Path:       "/home/dev",
		DeployRoot: "/home/dev",
		Kind:       registry.ConsumerContainer,
	}
	if err := reg.RegisterConsumer(c); err != nil {
		t.Fatalf("register consumer: %v", err)
	}

	// Register OS tool entries for each OS config type
	osEntries := []struct {
		name string
		typ  registry.CapabilityType
	}{
		{"zshrc", registry.TypeZshConfig},
		{"zprofile", registry.TypeZshConfig},
		{"init", registry.TypeNvimConfig},
		{"wezterm", registry.TypeWeztermConfig},
		{"mux-server", registry.TypeWeztermConfig},
		{"p10k.zsh", registry.TypeP10kConfig},
		{"config", registry.TypeHelixConfig},
	}
	for _, e := range osEntries {
		entry := registry.Entry{
			ID:           uuid.New(),
			SourceID:     src.ID,
			Name:         e.name,
			Type:         e.typ,
			ComposeMode:  registry.ModeOverride,
		}
		if err := reg.RegisterEntry(entry); err != nil {
			t.Fatalf("register entry %s: %v", e.name, err)
		}
	}

	if err := generateDefaultSlots(reg, c); err != nil {
		t.Fatalf("generateDefaultSlots: %v", err)
	}

	slots, err := reg.ResolveSlots(c.ID)
	if err != nil {
		t.Fatalf("resolve slots: %v", err)
	}

	// Verify expected slots for container consumer
	wantSlots := map[string]string{
		"zshrc":      ".zshrc",
		"zprofile":   ".zprofile",
		"init":       ".config/nvim/init.lua",
		"wezterm":    ".config/wezterm/wezterm.lua",
		"mux-server": ".config/wezterm/mux-server.lua",
		"p10k.zsh":   ".p10k.zsh",
		"config":     ".config/helix/config.toml",
	}

	if len(slots) != len(wantSlots) {
		t.Errorf("expected %d slots, got %d", len(wantSlots), len(slots))
		for _, s := range slots {
			t.Logf("  slot: %s -> %s", s.Name, s.DestPath)
		}
	}

	for _, s := range slots {
		wantDest, ok := wantSlots[s.Name]
		if !ok {
			t.Errorf("unexpected slot %q", s.Name)
			continue
		}
		if s.DestPath != wantDest {
			t.Errorf("slot %q: dest = %q, want %q", s.Name, s.DestPath, wantDest)
		}
	}
}

func TestGenerateDefaultSlots_RepoSkipsOSTypes(t *testing.T) {
	reg := registry.NewMemRegistry()

	src := registry.Source{
		ID:    uuid.New(),
		Alias: "carrel-omp",
		Path:  "/workspace/carrel/omp",
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	if err := reg.RegisterSource(src); err != nil {
		t.Fatalf("register source: %v", err)
	}

	c := registry.Consumer{
		ID:         uuid.New(),
		Alias:      "test-repo",
		Path:       "/tmp/test-repo",
		DeployRoot: "/tmp/test-repo/.omp",
		Kind:       registry.ConsumerRepo,
	}
	if err := reg.RegisterConsumer(c); err != nil {
		t.Fatalf("register consumer: %v", err)
	}

	// Register one OS entry and one OMP entry
	osEntry := registry.Entry{
		ID:          uuid.New(),
		SourceID:    src.ID,
		Name:        "zshrc",
		Type:        registry.TypeZshConfig,
		ComposeMode: registry.ModeOverride,
	}
	if err := reg.RegisterEntry(osEntry); err != nil {
		t.Fatalf("register os entry: %v", err)
	}
	ompEntry := registry.Entry{
		ID:          uuid.New(),
		SourceID:    src.ID,
		Name:        "test-rule",
		Type:        registry.TypeRule,
		ComposeMode: registry.ModeOverride,
	}
	if err := reg.RegisterEntry(ompEntry); err != nil {
		t.Fatalf("register omp entry: %v", err)
	}

	if err := generateDefaultSlots(reg, c); err != nil {
		t.Fatalf("generateDefaultSlots: %v", err)
	}

	slots, err := reg.ResolveSlots(c.ID)
	if err != nil {
		t.Fatalf("resolve slots: %v", err)
	}

	for _, s := range slots {
		if s.Name == "zshrc" {
			t.Errorf("repo consumer should not have OS slots, got %q -> %s", s.Name, s.DestPath)
		}
	}

	// Should have at least the rule slot
	found := false
	for _, s := range slots {
		if s.Name == "test-rule" {
			found = true
			break
		}
	}
	if !found {
		t.Error("repo consumer missing rule slot")
	}
}
