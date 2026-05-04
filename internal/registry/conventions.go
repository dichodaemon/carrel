package registry

import "fmt"

// Convention defines the on-disk layout for a capability type.
type Convention struct {
	// Dir is the subdirectory within a source where this type's files live.
	// Empty string means the source root.
	Dir string
	// FileName returns the filename for a given entry name.
	FileName func(name string) string
	// DefaultPrimitive is the composition primitive for this type when not overridden.
	DefaultPrimitive Primitive
	// IsSingleton is true when only one file of this type exists per source
	// (e.g., APPEND_SYSTEM.md, zshrc).
	IsSingleton bool
	// SingletonName is the fixed filename for singleton types.
	SingletonName string
}

// Conventions maps each capability type to its on-disk convention.
// This is the authoritative table referenced by composer/conventions.go
// and authoring CRUD operations.
var Conventions = map[CapabilityType]Convention{
	TypeRule: {
		Dir:              "rules",
		FileName:         func(name string) string { return name + ".md" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeSkill: {
		Dir:              "skills",
		FileName:         func(name string) string { return name + "/SKILL.md" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeCommand: {
		Dir:              "commands",
		FileName:         func(name string) string { return name + ".md" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeExtension: {
		Dir:              "extensions",
		FileName:         func(name string) string { return name + "/" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeAgent: {
		Dir:              "agents",
		FileName:         func(name string) string { return name + "/" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeTool: {
		Dir:              "tools",
		FileName:         func(name string) string { return name + "/" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeHook: {
		Dir:              "hooks",
		FileName:         func(name string) string { return name + ".md" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypePrompt: {
		Dir:              "prompts",
		FileName:         func(name string) string { return name + ".md" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeInstruction: {
		Dir:              "instructions",
		FileName:         func(name string) string { return name + ".md" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeContextFile: {
		Dir:              "",
		FileName:         func(name string) string { return name + ".md" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeAppendSystem: {
		Dir:              "",
		FileName:         func(name string) string { return name },
		DefaultPrimitive: PrimitiveConcatenation,
		IsSingleton:      true,
		SingletonName:    "APPEND_SYSTEM.md",
	},
	TypeZshConfig: {
		Dir:              "config/zsh",
		FileName:         func(name string) string { return name + ".zsh" },
		DefaultPrimitive: PrimitiveConcatenation,
	},
	TypeNvimConfig: {
		Dir:              "config/nvim",
		FileName:         func(name string) string { return name + ".lua" },
		DefaultPrimitive: PrimitiveConcatenation,
	},
	TypeWeztermConfig: {
		Dir:              "config/wezterm",
		FileName:         func(name string) string { return name + ".lua" },
		DefaultPrimitive: PrimitiveOverride,
	},
	TypeP10kConfig: {
		Dir:              "config/zsh",
		FileName:         func(name string) string { return name },
		DefaultPrimitive: PrimitiveOverride,
		IsSingleton:      true,
		SingletonName:    "p10k.zsh",
	},
}

// RelativePath returns the conventional relative path for an entry
// given its type and name.
func RelativePath(typ CapabilityType, name string) (string, error) {
	conv, ok := Conventions[typ]
	if !ok {
		return "", fmt.Errorf("unknown capability type %d", typ)
	}
	rel := conv.FileName(name)
	if conv.Dir != "" {
		rel = conv.Dir + "/" + rel
	}
	return rel, nil
}
