package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dichodaemon/carrel/internal/registry"
)

func slotGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <consumer-alias>",
		Short: "Generate .carrel/slots.yml from current registry state",
		Long: `Read the consumer's current slot definitions and entry-to-slot links
from the registry and write them to .carrel/slots.yml in the consumer root.

Idempotent — re-running produces identical output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			consumerAlias := args[0]
			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := reg.ResolveConsumer(consumerAlias)
			if err != nil {
				return fmt.Errorf("consumer %q: %w", consumerAlias, err)
			}

			sf, err := registry.GenerateSlotsFile(reg, consumerAlias)
			if err != nil {
				return err
			}

			data, err := yaml.Marshal(sf)
			if err != nil {
				return fmt.Errorf("marshal slots.yml: %w", err)
			}

			carrelDir := filepath.Join(c.Path, ".carrel")
			if err := os.MkdirAll(carrelDir, 0755); err != nil {
				return fmt.Errorf("create .carrel dir: %w", err)
			}

			outPath := filepath.Join(carrelDir, "slots.yml")
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("write slots.yml: %w", err)
			}

			fmt.Printf("generated: %s\n", outPath)
			return nil
		},
	}

	return cmd
}
