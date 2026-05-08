package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/authoring"
	"github.com/dichodaemon/carrel/internal/registry"
)

var capabilityTypes = []struct {
	name string
	typ  registry.CapabilityType
}{
	{"rule", registry.TypeRule},
	{"skill", registry.TypeSkill},
	{"command", registry.TypeCommand},
	{"extension", registry.TypeExtension},
	{"agent", registry.TypeAgent},
	{"tool", registry.TypeTool},
	{"hook", registry.TypeHook},
	{"prompt", registry.TypePrompt},
	{"instruction", registry.TypeInstruction},
	{"context-file", registry.TypeContextFile},
	{"append-system", registry.TypeAppendSystem},
	{"zsh", registry.TypeZshConfig},
	{"nvim", registry.TypeNvimConfig},
	{"wezterm", registry.TypeWeztermConfig},
	{"p10k", registry.TypeP10kConfig},
}



// resolveEntryFromArgs parses positional args to find an entry.
// If the first arg contains two colons, it's parsed as source:type:name.
// Otherwise args are (type, name). Returns the entry and its source.
func resolveEntryFromArgs(reg registry.Registry, args []string) (registry.Entry, registry.Source, error) {
	// Check for compound ID: source:type:name
	if len(args) >= 1 && strings.Count(args[0], ":") >= 2 {
		ref := args[0]
		sourceAlias, typ, name, err := registry.ParseEntryRef(ref)
		if err != nil {
			return registry.Entry{}, registry.Source{}, err
		}
		return authoring.FindEntryBySource(reg, sourceAlias, typ, name)
	}

	// Traditional (type, name)
	if len(args) < 2 {
		return registry.Entry{}, registry.Source{}, fmt.Errorf("type and name required")
	}
	typeName := args[0]
	name := args[1]
	typ, ok := lookupType(typeName)
	if !ok {
		return registry.Entry{}, registry.Source{}, fmt.Errorf("unknown type %q; run 'carrel config types' to see available types", typeName)
	}
	return authoring.FindEntry(reg, typ, name)
}
func crudAddCmd() *cobra.Command {
	var sourceAlias string
	var content string
	var filePath string

	cmd := &cobra.Command{
		Use:   "add <type> <name>",
		Short: "Add a configuration entry",
		Long: `Add a configuration entry to a registered source.

The entry file is created at the conventional path for the given type within
the source directory and registered in the carrel registry.

Examples:
  carrel config add rule no-push-master --source=carrel-omp --content="Never push to master"
  carrel config add skill validate --source=carrel-omp --file=./validate.md
  echo "content" | carrel config add hook pre-commit --source=carrel-omp

Available types — run 'carrel config types' to see all types with descriptions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel config add <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel config types' to see available types", typeName)
			}

			var data []byte
			if content != "" {
				data = []byte(content)
			} else if filePath != "" {
				var err error
				data, err = os.ReadFile(filePath)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("--content, --file, or stdin required")
			}

			reg := mustOpenRegistry()
			defer reg.Close()
			_, err := authoring.AddEntry(reg, typ, name, sourceAlias, data)
			return err
		},
	}

	cmd.Flags().StringVar(&sourceAlias, "source", "", "Source alias (required)")
	cmd.Flags().StringVar(&content, "content", "", "Entry content")
	cmd.Flags().StringVar(&filePath, "file", "", "Read content from file")
	return cmd
}

func crudListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <type>",
		Short: "List configuration entries by type",
		Long: `List all registered entries of a given type.

Shows each entry's name, source alias, and relative path.

Examples:
  carrel config list rule
  carrel config list skill

Available types — run 'carrel config types' to see all types with descriptions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("type required: carrel config list <type>")
			}
			typeName := args[0]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel config types' to see available types", typeName)
			}

			reg := mustOpenRegistry()
			defer reg.Close()

			sources, err := reg.ListSources()
			if err != nil {
				return err
			}
			var sourceIDs []uuid.UUID
			for _, s := range sources {
				sourceIDs = append(sourceIDs, s.ID)
			}

			sourceAlias := make(map[uuid.UUID]string)
			for _, s := range sources {
				sourceAlias[s.ID] = s.Alias
			}

			entries, err := reg.ResolveEntries(sourceIDs)
			if err != nil {
				return err
			}

			for _, e := range entries {
				if e.Type == typ {
					alias := sourceAlias[e.SourceID]
					fmt.Printf("%-30s %-20s %s\n", e.Name, alias, e.RelativePath)
				}
			}
			return nil
		},
	}
}

func crudRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <type> <name>",
		Short: "Remove a configuration entry",
		Long: `Remove a configuration entry from the registry and delete its file from disk.

Accepts either traditional (type, name) or compound ID (source:type:name).

Examples:
  carrel config rm rule no-push-master
  carrel config rm secondary-configs:append-system:APPEND_SYSTEM.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()
			entry, source, err := resolveEntryFromArgs(reg, args)
			if err != nil {
				return err
			}
			filePath := filepath.Join(source.Path, entry.RelativePath)
			os.Remove(filePath)
			return reg.RemoveEntry(entry.ID)
		},
	}
}

func crudViewCmd() *cobra.Command {
	var metaOnly bool
	var contentOnly bool

	cmd := &cobra.Command{
		Use:   "view <type> <name> or <source:type:name>",
		Short: "View a configuration entry",
		Long: `Show registry metadata and file content for a configuration entry.

By default both metadata and content are shown. Use --meta-only or
--content-only to filter.

Accepts either traditional (type, name) or compound ID (source:type:name).

Examples:
  carrel config view rule no-push-master
  carrel config view rule no-push-master --meta-only
  carrel config view secondary-configs:append-system:APPEND_SYSTEM.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()
			entry, source, err := resolveEntryFromArgs(reg, args)
			if err != nil {
				return err
			}

		showBoth := !metaOnly && !contentOnly
		typeName := typeToNameCRUD(entry.Type)

		if showBoth || !contentOnly {
			fmt.Printf("Name: %s\nType: %s\nPath: %s\nHash: %d\n",
				entry.Name, typeName, entry.RelativePath, entry.ContentHash)
		}

		if showBoth || !metaOnly {
			fmt.Println("\n--- Content ---")
			data, err := os.ReadFile(filepath.Join(source.Path, entry.RelativePath))
			if err == nil {
				fmt.Println(string(data))
				return nil
		}
		fmt.Println("(file not found on disk)")
	}
	return nil
		},
	}

	cmd.Flags().BoolVar(&metaOnly, "meta-only", false, "Show metadata only")
	cmd.Flags().BoolVar(&contentOnly, "content-only", false, "Show content only")
	return cmd
}

func crudEditCmd() *cobra.Command {
	var content string
	var filePath string

	cmd := &cobra.Command{
		Use:   "edit <type> <name> or <source:type:name>",
		Short: "Edit a configuration entry",
		Long: `Overwrite the content of a configuration entry's file and update the registry hash.

Accepts either traditional (type, name) or compound ID (source:type:name).

Examples:
  carrel config edit rule no-push-master --content="Updated rule content"
  carrel config edit skill validate --file=./updated-validate.md
  carrel config edit secondary-configs:append-system:APPEND_SYSTEM.md --content="..."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var data []byte
			if content != "" {
				data = []byte(content)
			} else if filePath != "" {
				var err error
				data, err = os.ReadFile(filePath)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("--content or --file required")
			}

			reg := mustOpenRegistry()
			defer reg.Close()
			entry, source, err := resolveEntryFromArgs(reg, args)
			if err != nil {
				return err
			}
			p := filepath.Join(source.Path, entry.RelativePath)
			if err := os.WriteFile(p, data, 0644); err != nil {
				return fmt.Errorf("write file: %w", err)
			}
			return reg.UpdateEntryMeta(entry.ID, registry.MetaUpdates{})
		},
	}

	cmd.Flags().StringVar(&content, "content", "", "New content")
	cmd.Flags().StringVar(&filePath, "file", "", "Read content from file")
	return cmd
}

func crudUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <type> <name> or <source:type:name>",
		Short: "Update entry metadata",
		Long: `Update metadata for a configuration entry.

Accepts either traditional (type, name) or compound ID (source:type:name).

Currently a placeholder — compose mode is set by convention during entry scan.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()
			entry, _, err := resolveEntryFromArgs(reg, args)
			if err != nil {
				return err
			}
			_ = entry
			return nil
		},
	}

	return cmd
}

func crudRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <type> <old-name> <new-name>",
		Short: "Rename a configuration entry",
		Long: `Rename a configuration entry, moving its file and updating the registry.
The entry's UUID is preserved.

Examples:
  carrel config rename rule old-name new-name
  carrel config rename carrel-omp:append-system:old-name new-name`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			// Compound ID: args[0] = source:type:oldName, args[1] = newName
			if len(args) >= 2 && strings.Count(args[0], ":") >= 2 {
				srcAlias, typ, name, err := registry.ParseEntryRef(args[0])
				if err != nil {
					return err
				}
				entry, source, err := authoring.FindEntryBySource(reg, srcAlias, typ, name)

				if err != nil {
					return err
				}
				return renameEntryBySource(reg, entry, source, args[1])
			}

			// Traditional: args[0] = type, args[1] = oldName, args[2] = newName
			if len(args) < 3 {
				return fmt.Errorf("type, old name, and new name required")
			}
			typeName := args[0]
			oldName := args[1]
			newName := args[2]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
			}
			return authoring.RenameEntry(reg, typ, oldName, newName)
		},
	}
}

func lookupType(name string) (registry.CapabilityType, bool) {
	for _, ct := range capabilityTypes {
		if ct.name == name {
			return ct.typ, true
		}
	}
	return 0, false
}

func typeToNameCRUD(typ registry.CapabilityType) string {
	for _, ct := range capabilityTypes {
		if ct.typ == typ {
			return ct.name
		}
	}
	return fmt.Sprintf("type-%d", typ)
}

func renameEntryBySource(reg registry.Registry, entry registry.Entry, source registry.Source, newName string) error {
	oldPath := filepath.Join(source.Path, entry.RelativePath)
	newRelPath := entry.RelativePath[:len(entry.RelativePath)-len(entry.Name)] + newName
	newPath := filepath.Join(source.Path, newRelPath)
	os.MkdirAll(filepath.Dir(newPath), 0755)
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}
	return authoring.RenameEntry(reg, entry.Type, entry.Name, newName)
}
