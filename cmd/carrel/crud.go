package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/authoring"
	"github.com/dichodaemon/carrel/internal/registry"
)

// addCRUDCommands adds per-type CRUD subcommands to the root command.
func addCRUDCommands(root *cobra.Command) {
	types := []struct {
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

	for _, t := range types {
		cmd := &cobra.Command{Use: t.name, Short: fmt.Sprintf("Manage %s entries", t.name)}

		cmd.AddCommand(crudAddCmd(t.name, t.typ))
		cmd.AddCommand(crudListCmd(t.name, t.typ))
		cmd.AddCommand(crudRmCmd(t.name, t.typ))
		cmd.AddCommand(crudViewCmd(t.name, t.typ))
		cmd.AddCommand(crudEditCmd(t.name, t.typ))
		cmd.AddCommand(crudUpdateCmd(t.name, t.typ))
		cmd.AddCommand(crudRenameCmd(t.name, t.typ))

		root.AddCommand(cmd)
	}
}

func crudAddCmd(typeName string, typ registry.CapabilityType) *cobra.Command {
	var sourceAlias string
	var content string
	var filePath string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: fmt.Sprintf("Add a %s entry", typeName),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("name required")
			}
			name := args[0]

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

func crudListCmd(typeName string, typ registry.CapabilityType) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s entries", typeName),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			entries, err := reg.ResolveEntries(sourceIDs)
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.Type == typ {
					fmt.Printf("%s/%s (%s)\n", e.Name, e.RelativePath, e.SourceID)
				}
			}
			return nil
		},
	}
}

func crudRmCmd(typeName string, typ registry.CapabilityType) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: fmt.Sprintf("Remove a %s entry", typeName),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("name required")
			}
			reg := mustOpenRegistry()
			defer reg.Close()
			return authoring.RemoveEntry(reg, typ, args[0])
		},
	}
}

func crudViewCmd(typeName string, typ registry.CapabilityType) *cobra.Command {
	return &cobra.Command{
		Use:   "view <name>",
		Short: fmt.Sprintf("View a %s entry", typeName),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("name required")
			}
			reg := mustOpenRegistry()
			defer reg.Close()
			// Resolve entry by name+type
			sources, _ := reg.ListSources()
			var sourceIDs []uuid.UUID
			for _, s := range sources {
				sourceIDs = append(sourceIDs, s.ID)
			}
			entries, _ := reg.ResolveEntries(sourceIDs)
			for _, e := range entries {
				if e.Type == typ && e.Name == args[0] {
					fmt.Printf("Name: %s\nType: %s\nPath: %s\nHash: %d\nFinal: %v\n",
						e.Name, typeName, e.RelativePath, e.ContentHash, e.Final)
					return nil
				}
			}
			return fmt.Errorf("%s %q not found", typeName, args[0])
		},
	}
}

func crudEditCmd(typeName string, typ registry.CapabilityType) *cobra.Command {
	var content string
	var filePath string

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: fmt.Sprintf("Edit a %s entry", typeName),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("name required")
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
			return authoring.EditEntry(reg, typ, args[0], data)
		},
	}

	cmd.Flags().StringVar(&content, "content", "", "New content")
	cmd.Flags().StringVar(&filePath, "file", "", "Read content from file")
	return cmd
}

func crudUpdateCmd(typeName string, typ registry.CapabilityType) *cobra.Command {
	var final bool

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: fmt.Sprintf("Update %s metadata", typeName),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("name required")
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
				if e.Type == typ && e.Name == args[0] {
					updates := registry.MetaUpdates{}
					if cmd.Flags().Changed("final") {
						updates.Final = &final
					}
					return reg.UpdateEntryMeta(e.ID, updates)
				}
			}
			return fmt.Errorf("%s %q not found", typeName, args[0])
		},
	}

	cmd.Flags().BoolVar(&final, "final", false, "Set final flag")
	return cmd
}

func crudRenameCmd(typeName string, typ registry.CapabilityType) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: fmt.Sprintf("Rename a %s entry", typeName),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("old and new names required")
			}
			reg := mustOpenRegistry()
			defer reg.Close()
			return authoring.RenameEntry(reg, typ, args[0], args[1])
		},
	}
}
