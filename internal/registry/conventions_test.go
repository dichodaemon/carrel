package registry_test

import (
	"testing"

	"github.com/dichodaemon/carrel/internal/registry"
)

func TestConventionsCoverAllTypes(t *testing.T) {
	types := []registry.CapabilityType{
		registry.TypeRule,
		registry.TypeSkill,
		registry.TypeCommand,
		registry.TypeExtension,
		registry.TypeAgent,
		registry.TypeTool,
		registry.TypeHook,
		registry.TypePrompt,
		registry.TypeInstruction,
		registry.TypeContextFile,
		registry.TypeAppendSystem,
		registry.TypeZshConfig,
		registry.TypeNvimConfig,
		registry.TypeWeztermConfig,
		registry.TypeP10kConfig,
	}

	for _, typ := range types {
		_, ok := registry.Conventions[typ]
		if !ok {
			t.Errorf("type %d missing from Conventions", typ)
		}
	}
}

func TestRelativePath(t *testing.T) {
	tests := []struct {
		typ  registry.CapabilityType
		name string
		want string
	}{
		{registry.TypeRule, "no-push-master", "rules/no-push-master.md"},
		{registry.TypeSkill, "grill-me", "skills/grill-me/SKILL.md"},
		{registry.TypeCommand, "build", "commands/build.md"},
		{registry.TypeExtension, "my-ext", "extensions/my-ext/"},
		{registry.TypeAgent, "my-agent", "agents/my-agent/"},
		{registry.TypeTool, "my-tool", "tools/my-tool/"},
		{registry.TypeHook, "pre-commit", "hooks/pre-commit.md"},
		{registry.TypePrompt, "review", "prompts/review.md"},
		{registry.TypeInstruction, "setup", "instructions/setup.md"},
		{registry.TypeContextFile, "AGENTS", "AGENTS.md"},
		{registry.TypeAppendSystem, "APPEND_SYSTEM.md", "APPEND_SYSTEM.md"},
		{registry.TypeZshConfig, "zshrc", "config/zsh/zshrc.zsh"},
		{registry.TypeNvimConfig, "init", "config/nvim/init.lua"},
		{registry.TypeWeztermConfig, "mux-server", "config/wezterm/mux-server.lua"},
		{registry.TypeP10kConfig, "p10k.zsh", "config/zsh/p10k.zsh"},
	}

	for _, tt := range tests {
		got, err := registry.RelativePath(tt.typ, tt.name)
		if err != nil {
			t.Errorf("RelativePath(%d, %q): %v", tt.typ, tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("RelativePath(%d, %q) = %q, want %q", tt.typ, tt.name, got, tt.want)
		}
	}
}

func TestAppendSystemConcatenation(t *testing.T) {
	conv := registry.Conventions[registry.TypeAppendSystem]
	if conv.DefaultPrimitive != registry.PrimitiveConcatenation {
		t.Error("APPEND_SYSTEM.md must use concatenation primitive")
	}
	if !conv.IsSingleton {
		t.Error("APPEND_SYSTEM.md must be a singleton")
	}
}

func TestDefaultPrimitives(t *testing.T) {
	// OMP config types use override by default
	overrideTypes := []registry.CapabilityType{
		registry.TypeRule,
		registry.TypeSkill,
		registry.TypeCommand,
		registry.TypeHook,
		registry.TypePrompt,
		registry.TypeInstruction,
	}
	for _, typ := range overrideTypes {
		if registry.Conventions[typ].DefaultPrimitive != registry.PrimitiveOverride {
			t.Errorf("type %d should default to override", typ)
		}
	}

	// Concatenation types
	concatTypes := []registry.CapabilityType{
		registry.TypeAppendSystem,
		registry.TypeZshConfig,
		registry.TypeNvimConfig,
	}
	for _, typ := range concatTypes {
		if registry.Conventions[typ].DefaultPrimitive != registry.PrimitiveConcatenation {
			t.Errorf("type %d should default to concatenation", typ)
		}
	}
}
