package main

import (
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Type-aware configuration management",
		Long:  "Commands that operate on typed configuration entries (rules, skills, OS configs, etc.) using the capability type convention system.",
	}

	cmd.AddGroup(&cobra.Group{ID: "config-lifecycle", Title: "Lifecycle Commands:"})
	cmd.AddGroup(&cobra.Group{ID: "config-crud", Title: "Configuration CRUD:"})
	cmd.AddGroup(&cobra.Group{ID: "config-query", Title: "Query Commands:"})

	// Lifecycle
	sc := scanCmd()
	sc.GroupID = "config-lifecycle"
	cmd.AddCommand(sc)

	// CRUD
	for _, f := range []func() *cobra.Command{crudAddCmd, crudListCmd, crudRmCmd, crudViewCmd, crudRenameCmd} {
		c := f()
		c.GroupID = "config-crud"
		cmd.AddCommand(c)
	}

	// Query
	tc := typesCmd()
	tc.GroupID = "config-query"
	cmd.AddCommand(tc)

	return cmd
}
