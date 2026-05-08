package main

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/registry"
	"github.com/dichodaemon/carrel/internal/scanner"
)

func registerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register consumers and sources",
		Long:  "Register new consumers (target repos) and configuration sources in the carrel registry.",
	}
	cmd.AddCommand(registerConsumerCmd())
	cmd.AddCommand(registerSourceCmd())
	return cmd
}

func registerConsumerCmd() *cobra.Command {
	var path string
	var deployRoot string
	var kind string

	cmd := &cobra.Command{
		Use:   "consumer <alias>",
		Short: "Register a consumer (target repo)",
		Long: `Register a repository as a carrel consumer. Idempotent — safe to re-run.

Examples:
  carrel register consumer core-stack --path=/workspace/core-stack
  carrel register consumer core-stack --path=/workspace/core-stack --deploy-root=/workspace/core-stack/.omp`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if path == "" {
				return fmt.Errorf("--path is required")
			}
			if deployRoot == "" {
				deployRoot = path + "/.omp"
			}

			var ck registry.ConsumerKind
			switch kind {
			case "repo":
				ck = registry.ConsumerRepo
			case "container":
				ck = registry.ConsumerContainer
			case "host":
				ck = registry.ConsumerHost
			default:
				return fmt.Errorf("unknown kind %q; must be repo, container, or host", kind)
			}

			reg := mustOpenRegistry()
			defer reg.Close()

			// Check if already registered by alias or path; reuse ID if so.
			id := uuid.New()
			if existing, err := reg.ResolveConsumer(alias); err == nil {
				id = existing.ID
			} else if existing, err := reg.ResolveConsumer(path); err == nil {
				id = existing.ID
			}

			c := registry.Consumer{
				ID:         id,
				Alias:      alias,
				Path:       path,
				DeployRoot: deployRoot,
				Kind:       ck,
			}
			if err := reg.RegisterConsumer(c); err != nil {
				return fmt.Errorf("register consumer: %w", err)
			}

			// Apply .carrel/slots.yml if present
			slotsPath := filepath.Join(path, ".carrel", "slots.yml")
			warnings, err := registry.ApplySlotsFile(reg, c, slotsPath)
			if err != nil {
				// File read error (not missing) — warn but don't fail
				fmt.Printf("  warning: slots.yml read error: %v\n", err)
			}
			for _, w := range warnings {
				fmt.Printf("  warning: slots.yml: %s\n", w)
			}

			fmt.Printf("registered consumer: %s (%s)\n", alias, path)
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Absolute path to the consumer repo (required)")
	cmd.Flags().StringVar(&deployRoot, "deploy-root", "", "Deployment root (default: <path>/.omp)")
	cmd.Flags().StringVar(&kind, "kind", "repo", "Consumer kind: repo, container, host")
	return cmd
}

func registerSourceCmd() *cobra.Command {
	var path string
	var scope string
	var consumerAlias string

	cmd := &cobra.Command{
		Use:   "source <alias>",
		Short: "Register a configuration source",
		Long: `Register a directory as a carrel configuration source. Idempotent — safe to re-run.

When --consumer is provided, the source is linked to the consumer, entries are
scanned, and default slots are generated in one pass.

Examples:
  carrel register source secondary-configs --path=/workspace/secondary-configs/omp --scope=target --consumer=core-stack
  carrel register source my-source --path=/workspace/my-source/omp --scope=universal`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			if path == "" {
				return fmt.Errorf("--path is required")
			}
			if scope == "" {
				return fmt.Errorf("--scope is required (universal or target)")
			}

			var ss registry.SourceScope
			switch scope {
			case "universal":
				ss = registry.ScopeUniversal
			case "target":
				ss = registry.ScopeTargetSpecific
			default:
				return fmt.Errorf("unknown scope %q; must be universal or target", scope)
			}

			reg := mustOpenRegistry()
			defer reg.Close()

			// Check if already registered; reuse ID if so.
			id := uuid.New()
			sources, _ := reg.ListSources()
			for _, s := range sources {
				if s.Alias == alias {
					id = s.ID
					break
				}
			}

			src := registry.Source{
				ID:    id,
				Alias: alias,
				Path:  path,
				Scope: ss,
				Kind:  registry.SourceGitBacked,
			}
			if err := reg.RegisterSource(src); err != nil {
				return fmt.Errorf("register source: %w", err)
			}
			fmt.Printf("registered source: %s (%s, scope=%s)\n", alias, path, scope)

			// Link to consumer if specified.
			if consumerAlias != "" {
				c, err := reg.ResolveConsumer(consumerAlias)
				if err != nil {
					return fmt.Errorf("resolve consumer %q: %w", consumerAlias, err)
				}

				if err := reg.LinkConsumerSource(c.ID, id); err != nil {
					return fmt.Errorf("link source to consumer: %w", err)
				}
				fmt.Printf("linked source %s to consumer %s\n", alias, consumerAlias)

				// Scan entries.
				results, err := scanner.ScanSource(reg, src)
				if err != nil {
					return fmt.Errorf("scan source: %w", err)
				}
				for _, r := range results {
					if r.Action == "error" {
						fmt.Printf("  %-12s %-15s %-20s error: %v\n", r.Action, r.Type, r.Name, r.Error)
					} else {
						fmt.Printf("  %-12s %-15s %s\n", r.Action, r.Type, r.Name)
					}
				}

				// Apply .carrel/slots.yml if present, otherwise generate defaults
				slotsPath := filepath.Join(c.Path, ".carrel", "slots.yml")
				warnings, err := registry.ApplySlotsFile(reg, c, slotsPath)
				if err != nil {
					fmt.Printf("  warning: slots.yml read error: %v\n", err)
					if err := generateDefaultSlots(reg, c); err != nil {
						fmt.Printf("  warning: slot generation: %v\n", err)
					}
				} else {
					for _, w := range warnings {
						fmt.Printf("  warning: slots.yml: %s\n", w)
					}
					fmt.Printf("applied slots.yml for %s\n", consumerAlias)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Absolute path to the source directory (required)")
	cmd.Flags().StringVar(&scope, "scope", "", "Source scope: universal or target (required)")
	cmd.Flags().StringVar(&consumerAlias, "consumer", "", "Consumer alias to link, scan, and generate slots for")
	return cmd
}
