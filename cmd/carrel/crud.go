package main

import (
	"fmt"
	"os"
	"path/filepath"

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
  carrel add rule no-push-master --source=carrel-omp --content="Never push to master"
  carrel add skill validate --source=carrel-omp --file=./validate.md
  echo "content" | carrel add hook pre-commit --source=carrel-omp

Available types — run 'carrel types' to see all types with descriptions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel add <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel types' to see available types", typeName)
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
  carrel list rule
  carrel list skill

Available types — run 'carrel types' to see all types with descriptions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("type required: carrel list <type>")
			}
			typeName := args[0]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel types' to see available types", typeName)
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

Examples:
  carrel rm rule no-push-master
  carrel rm skill validate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel rm <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel types' to see available types", typeName)
			}

			reg := mustOpenRegistry()
			defer reg.Close()
			return authoring.RemoveEntry(reg, typ, name)
		},
	}
}

func crudViewCmd() *cobra.Command {
	var metaOnly bool
	var contentOnly bool

	cmd := &cobra.Command{
		Use:   "view <type> <name>",
		Short: "View a configuration entry",
		Long: `Show registry metadata and file content for a configuration entry.

By default both metadata and content are shown. Use --meta-only or
--content-only to filter.

Examples:
  carrel view rule no-push-master
  carrel view rule no-push-master --meta-only
  carrel view skill validate --content-only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel view <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel types' to see available types", typeName)
			}

			reg := mustOpenRegistry()
			defer reg.Close()

			sources, _ := reg.ListSources()
			var sourceIDs []uuid.UUID
			for _, s := range sources {
				sourceIDs = append(sourceIDs, s.ID)
			}
			entries, _ := reg.ResolveEntries(sourceIDs)
			for _, e := range entries {
				if e.Type == typ && e.Name == name {
					showBoth := !metaOnly && !contentOnly

					if showBoth || !contentOnly {
						fmt.Printf("Name: %s\nType: %s\nPath: %s\nHash: %d\nFinal: %v\n",
							e.Name, typeName, e.RelativePath, e.ContentHash, e.Final)
					}

					if showBoth || !metaOnly {
						fmt.Println("\n--- Content ---")
						for _, s := range sources {
							data, err := os.ReadFile(filepath.Join(s.Path, e.RelativePath))
							if err == nil {
								fmt.Println(string(data))
								return nil
							}
						}
						fmt.Println("(file not found on disk)")
					}
					return nil
				}
			}
			return fmt.Errorf("%s %q not found", typeName, name)
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
		Use:   "edit <type> <name>",
		Short: "Edit a configuration entry",
		Long: `Overwrite the content of a configuration entry's file and update the registry hash.

Examples:
  carrel edit rule no-push-master --content="Updated rule content"
  carrel edit skill validate --file=./updated-validate.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel edit <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel types' to see available types", typeName)
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
				return fmt.Errorf("--content or --file required")
			}

			reg := mustOpenRegistry()
			defer reg.Close()
			return authoring.EditEntry(reg, typ, name, data)
		},
	}

	cmd.Flags().StringVar(&content, "content", "", "New content")
	cmd.Flags().StringVar(&filePath, "file", "", "Read content from file")
	return cmd
}

func crudUpdateCmd() *cobra.Command {
	var final bool

	cmd := &cobra.Command{
		Use:   "update <type> <name>",
		Short: "Update entry metadata",
		Long: `Update metadata for a configuration entry without changing its content.

Currently supports setting the 'final' flag, which prevents deeper scopes
from overriding the entry during composition.

Examples:
  carrel update rule no-push-master --final=true
  carrel update skill validate --final=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel update <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel types' to see available types", typeName)
			}

			reg := mustOpenRegistry()
			defer reg.Close()

			sources, _ := reg.ListSources()
			var sourceIDs []uuid.UUID
			for _, s := range sources {
				sourceIDs = append(sourceIDs, s.ID)
			}
			entries, _ := reg.ResolveEntries(sourceIDs)
			for _, e := range entries {
				if e.Type == typ && e.Name == name {
					updates := registry.MetaUpdates{}
					if cmd.Flags().Changed("final") {
						updates.Final = &final
					}
					return reg.UpdateEntryMeta(e.ID, updates)
				}
			}
			return fmt.Errorf("%s %q not found", typeName, name)
		},
	}

	cmd.Flags().BoolVar(&final, "final", false, "Set final flag (prevents deeper scope override)")
	return cmd
}

func crudRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <type> <old-name> <new-name>",
		Short: "Rename a configuration entry",
		Long: `Rename a configuration entry, moving its file and updating the registry.
The entry's UUID is preserved.

Examples:
  carrel rename rule old-name new-name
  carrel rename skill old-validate new-validate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return fmt.Errorf("type, old name, and new name required: carrel rename <type> <old> <new>")
			}
			typeName := args[0]
			oldName := args[1]
			newName := args[2]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q; run 'carrel types' to see available types", typeName)
			}

			reg := mustOpenRegistry()
			defer reg.Close()
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
