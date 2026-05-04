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

// capabilityTypes lists all OMP capability types for CRUD.
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
}

// addCRUDCommands adds verb-first CRUD subcommands to the root command.
// Commands: carrel add <type> <name>, carrel list <type>, etc.
func addCRUDCommands(root *cobra.Command) {
	for _, f := range []func() *cobra.Command{crudAddCmd, crudListCmd, crudRmCmd, crudViewCmd, crudEditCmd, crudUpdateCmd, crudRenameCmd} {
		cmd := f()
		cmd.GroupID = "crud"
		root.AddCommand(cmd)
	}
}

func crudAddCmd() *cobra.Command {
	var sourceAlias string
	var content string
	var filePath string

	cmd := &cobra.Command{
		Use:   "add <type> <name>",
		Short: "Add a configuration entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel add <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
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

	cmd.Flags().StringVar(&sourceAlias, "source", "", "Source alias")
	cmd.Flags().StringVar(&content, "content", "", "Entry content")
	cmd.Flags().StringVar(&filePath, "file", "", "Read content from file")
	return cmd
}

func crudListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <type>",
		Short: "List configuration entries by type",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("type required: carrel list <type>")
			}
			typeName := args[0]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel rm <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
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
		Long:  "Shows registry metadata and file content. Use --meta-only or --content-only to filter.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel view <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel edit <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("type and name required: carrel update <type> <name>")
			}
			typeName := args[0]
			name := args[1]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
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

	cmd.Flags().BoolVar(&final, "final", false, "Set final flag")
	return cmd
}

func crudRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <type> <old-name> <new-name>",
		Short: "Rename a configuration entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return fmt.Errorf("type, old name, and new name required: carrel rename <type> <old> <new>")
			}
			typeName := args[0]
			oldName := args[1]
			newName := args[2]
			typ, ok := lookupType(typeName)
			if !ok {
				return fmt.Errorf("unknown type %q", typeName)
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
