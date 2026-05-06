package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// typeDescriptions maps type names to brief descriptions for help output.
var typeDescriptions = map[string]string{
	"rule":          "Rules that govern agent behavior (conventions, constraints, prohibitions)",
	"skill":         "Agent skills for specialized workflows",
	"command":       "Slash-command definitions",
	"extension":     "OMP extensions",
	"agent":         "Agent personality and model configurations",
	"tool":          "External tool integrations",
	"hook":          "Lifecycle hooks (pre/post session operations)",
	"prompt":        "Reusable prompt templates",
	"instruction":   "Instructional content injected into sessions",
	"context-file":  "Project-level context files (AGENTS.md, CLAUDE.md)",
	"append-system": "System prompt appendix content (concatenated across sources)",
}

func typesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List available configuration types",
		Long:  "Lists all configuration types that can be managed with CRUD commands (add, list, rm, view, edit, update, rename).",
		RunE: func(cmd *cobra.Command, args []string) error {
			maxLen := 0
			for _, ct := range capabilityTypes {
				if len(ct.name) > maxLen {
					maxLen = len(ct.name)
				}
			}
			for _, ct := range capabilityTypes {
				desc := typeDescriptions[ct.name]
				fmt.Printf("  %-*s  %s\n", maxLen, ct.name, desc)
			}
			return nil
		},
	}
}
