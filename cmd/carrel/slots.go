package main

import (
	"fmt"
	"os"
	"strings"


	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/registry"
)

func slotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slot",
		Short: "Manage output slots for consumers",
		Long:  "Commands for managing output slots that map configuration entries to deployment paths.",
	}

	cmd.AddCommand(slotAddCmd())
	cmd.AddCommand(slotListCmd())
	cmd.AddCommand(slotRmCmd())
	cmd.AddCommand(slotViewCmd())
	cmd.AddCommand(slotAddEntryCmd())
	cmd.AddCommand(slotRmEntryCmd())
	cmd.AddCommand(slotSyncCmd())
	cmd.AddCommand(slotGenerateCmd())

	return cmd
}

func slotAddCmd() *cobra.Command {
	var destPath string
	var consumerAlias string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new output slot",
		Long:  "Creates a new output slot for a consumer. The slot maps entries to a deployment destination path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := resolveSlotConsumer(reg, consumerAlias)
			if err != nil {
				return err
			}

			slot := registry.Slot{
				ID:         uuid.New(),
				ConsumerID: c.ID,
				Name:       args[0],
				DestPath:   destPath,
			}
			if err := reg.RegisterSlot(slot); err != nil {
				return fmt.Errorf("register slot: %w", err)
			}
			fmt.Printf("slot %q created for consumer %q\n", args[0], c.Alias)
			return nil
		},
	}
	cmd.Flags().StringVar(&destPath, "dest", "", "Destination path relative to consumer deploy root (required)")
	cmd.Flags().StringVar(&consumerAlias, "consumer", "", "Consumer alias (resolved from cwd if not set)")
	cmd.MarkFlagRequired("dest")
	return cmd
}

func slotListCmd() *cobra.Command {
	var consumerAlias string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List output slots",
		Long:  "Lists all output slots for a consumer, showing name, destination path, and number of linked entries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := resolveSlotConsumer(reg, consumerAlias)
			if err != nil {
				return err
			}

			slots, err := reg.ResolveSlots(c.ID)
			if err != nil {
				return fmt.Errorf("resolve slots: %w", err)
			}

			for _, s := range slots {
				entries, _ := reg.ResolveEntrySlots(s.ID)
				fmt.Printf("%-30s %-40s %d entries\n", s.Name, s.DestPath, len(entries))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&consumerAlias, "consumer", "", "Consumer alias (resolved from cwd if not set)")
	return cmd
}

func slotRmCmd() *cobra.Command {
	var consumerAlias string

	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove an output slot",
		Long:  "Removes an output slot and all its entry links. Does not delete deployed files.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := resolveSlotConsumer(reg, consumerAlias)
			if err != nil {
				return err
			}

			slots, err := reg.ResolveSlots(c.ID)
			if err != nil {
				return fmt.Errorf("resolve slots: %w", err)
			}
			for _, s := range slots {
				if s.Name == args[0] {
					if err := reg.RemoveSlot(s.ID); err != nil {
						return fmt.Errorf("remove slot: %w", err)
					}
					fmt.Printf("slot %q removed\n", args[0])
					return nil
				}
			}
			return fmt.Errorf("slot %q not found for consumer %q", args[0], c.Alias)
		},
	}
	cmd.Flags().StringVar(&consumerAlias, "consumer", "", "Consumer alias (resolved from cwd if not set)")
	return cmd
}

func slotViewCmd() *cobra.Command {
	var consumerAlias string

	cmd := &cobra.Command{
		Use:   "view <name>",
		Short: "View slot details and linked entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := resolveSlotConsumer(reg, consumerAlias)
			if err != nil {
				return err
			}

			slots, err := reg.ResolveSlots(c.ID)
			if err != nil {
				return fmt.Errorf("resolve slots: %w", err)
			}

			var slot *registry.Slot
			for _, s := range slots {
				if s.Name == args[0] {
					slot = &s
					break
				}
			}
			if slot == nil {
				return fmt.Errorf("slot %q not found for consumer %q", args[0], c.Alias)
			}

			fmt.Printf("Name:       %s\n", slot.Name)
			fmt.Printf("Dest:       %s\n", c.DeployRoot+"/"+slot.DestPath)
			if slot.ComposeMode != nil {
				fmt.Printf("Mode:       %d\n", *slot.ComposeMode)
			} else {
				fmt.Println("Mode:       (inherited from entries)")
			}

			entries, err := reg.ResolveEntrySlots(slot.ID)
			if err == nil && len(entries) > 0 {
				fmt.Println("\nLinked entries:")
				for _, se := range entries {
					fmt.Printf("  %s (priority=%d)\n", se.Name, se.Priority)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&consumerAlias, "consumer", "", "Consumer alias (resolved from cwd if not set)")
	return cmd
}

func slotAddEntryCmd() *cobra.Command {
	var consumerAlias string

	cmd := &cobra.Command{
		Use:   "add-entry <entry-compound-id> <slot-name>",
		Short: "Link an entry to a slot",
		Long:  "Links a configuration entry to an output slot. Entry compound ID format: <source-alias>:<type>:<name> (e.g., carrel-omp:rule:carrel-configuration).",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			entry, err := resolveEntryByCompoundID(reg, args[0])
			if err != nil {
				return err
			}

			c, err := resolveSlotConsumer(reg, consumerAlias)
			if err != nil {
				return err
			}

			slots, err := reg.ResolveSlots(c.ID)
			if err != nil {
				return fmt.Errorf("resolve slots: %w", err)
			}
			var slotID uuid.UUID
			for _, s := range slots {
				if s.Name == args[1] {
					slotID = s.ID
					break
				}
			}
			if slotID == uuid.Nil {
				return fmt.Errorf("slot %q not found for consumer %q", args[1], c.Alias)
			}

			// Use priority from scope
			priority := 0
			sources, _ := reg.ListSources()
			for _, src := range sources {
				if src.ID == entry.SourceID {
					priority = registry.ScopePriority[src.Scope]
					break
				}
			}

			if err := reg.LinkEntrySlot(entry.ID, slotID, priority); err != nil {
				return fmt.Errorf("link entry to slot: %w", err)
			}
			fmt.Printf("entry %q linked to slot %q (priority=%d)\n", args[0], args[1], priority)
			return nil
		},
	}
	cmd.Flags().StringVar(&consumerAlias, "consumer", "", "Consumer alias (resolved from cwd if not set)")
	return cmd
}

func slotRmEntryCmd() *cobra.Command {
	var consumerAlias string

	cmd := &cobra.Command{
		Use:   "rm-entry <entry-compound-id> <slot-name>",
		Short: "Unlink an entry from a slot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			entry, err := resolveEntryByCompoundID(reg, args[0])
			if err != nil {
				return err
			}

			c, err := resolveSlotConsumer(reg, consumerAlias)
			if err != nil {
				return err
			}

			slots, err := reg.ResolveSlots(c.ID)
			if err != nil {
				return fmt.Errorf("resolve slots: %w", err)
			}
			var slotID uuid.UUID
			for _, s := range slots {
				if s.Name == args[1] {
					slotID = s.ID
					break
				}
			}
			if slotID == uuid.Nil {
				return fmt.Errorf("slot %q not found for consumer %q", args[1], c.Alias)
			}

			if err := reg.UnlinkEntrySlot(entry.ID, slotID); err != nil {
				return fmt.Errorf("unlink entry from slot: %w", err)
			}
			fmt.Printf("entry %q unlinked from slot %q\n", args[0], args[1])
			return nil
		},
	}
	cmd.Flags().StringVar(&consumerAlias, "consumer", "", "Consumer alias (resolved from cwd if not set)")
	return cmd
}

// resolveSlotConsumer resolves a consumer from --consumer flag or cwd.
func resolveSlotConsumer(reg registry.Registry, alias string) (registry.Consumer, error) {
	if alias != "" {
		return reg.ResolveConsumer(alias)
	}
	cwd, _ := os.Getwd()
	gitRoot := findGitRoot(cwd)
	if gitRoot == "" {
		return registry.Consumer{}, fmt.Errorf("not in a git repository; use --consumer flag")
	}
	c, err := reg.ResolveConsumer(gitRoot)
	if err != nil {
		return registry.Consumer{}, fmt.Errorf("unregistered repo; use --consumer flag")
	}
	return c, nil
}

// resolveEntryByCompoundID resolves an entry from a compound ID: <source>:<type>:<name>
func resolveEntryByCompoundID(reg registry.Registry, compoundID string) (registry.Entry, error) {
	sources, err := reg.ListSources()
	if err != nil {
		return registry.Entry{}, err
	}
	sourceMap := make(map[string]registry.Source)
	for _, s := range sources {
		sourceMap[s.Alias] = s
	}

	parts := strings.SplitN(compoundID, ":", 3)
	if len(parts) != 3 {
		return registry.Entry{}, fmt.Errorf("invalid compound ID %q; format: <source>:<type>:<name>", compoundID)
	}
	sourceAlias, typeName, entryName := parts[0], parts[1], parts[2]

	src, ok := sourceMap[sourceAlias]
	if !ok {
		return registry.Entry{}, fmt.Errorf("source %q not found", sourceAlias)
	}

	typ, ok := lookupType(typeName)
	if !ok {
		return registry.Entry{}, fmt.Errorf("unknown type %q", typeName)
	}

	var sourceIDs []uuid.UUID
	for _, s := range sources {
		sourceIDs = append(sourceIDs, s.ID)
	}
	entries, _ := reg.ResolveEntries(sourceIDs)
	for _, e := range entries {
		if e.SourceID == src.ID && e.Type == typ && e.Name == entryName {
			return e, nil
		}
	}
	return registry.Entry{}, fmt.Errorf("entry %q not found", compoundID)
}
