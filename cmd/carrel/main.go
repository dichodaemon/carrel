package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	tea "charm.land/bubbletea/v2"

	"github.com/dichodaemon/carrel/internal/composer"
	"github.com/dichodaemon/carrel/internal/deployer"
	"github.com/dichodaemon/carrel/internal/registry"
	"github.com/dichodaemon/carrel/internal/scanner"
	"github.com/dichodaemon/carrel/internal/tui"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "carrel",
		Short: "Carrel — OMP and OS tool configuration manager",
	}

	rootCmd.AddGroup(&cobra.Group{ID: "lifecycle", Title: "Lifecycle Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: "crud", Title: "Configuration CRUD:"})
	rootCmd.AddGroup(&cobra.Group{ID: "query", Title: "Query Commands:"})

	// Lifecycle commands
	for _, f := range []func() *cobra.Command{bootstrapCmd, discoverCmd, migrateCmd, runCmd, osSetupCmd, hostSetupCmd} {
		cmd := f()
		cmd.GroupID = "lifecycle"
		rootCmd.AddCommand(cmd)
	}

	// Query commands
	for _, f := range []func() *cobra.Command{statusCmd, verifyCmd, dashboardCmd, typesCmd, sourcesCmd, deployedCmd, planCmd, dependentsCmd, feedsCmd, inspectCmd, traceCmd} {
		cmd := f()
		cmd.GroupID = "query"
		rootCmd.AddCommand(cmd)
	}

	addCRUDCommands(rootCmd)


	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func bootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Initialize the carrel registry",
		Long:  "Creates the registry at /workspace/.carrel/ with current schema. Registers carrel and folio consumers. Idempotent.",
		RunE:  runBootstrap,
	}
}

func discoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Scan workspace for git repos",
		Long:  "Scans /workspace for git repositories and reports their registration status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			repos, err := scanner.Discover(reg, "/workspace")
			if err != nil {
				return err
			}

			for _, r := range repos {
				status := "unregistered"
				if r.Registered {
					status = "registered"
				}
				flags := ""
				if r.HasCarrel {
					flags += " [opt-in]"
				}
				if r.HasCarula {
					flags += " [carula symlink]"
				}
				fmt.Printf("%-40s %s%s\n", r.Path, status, flags)
			}
			return nil
		},
	}
}

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Import carula configuration into registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			results, err := scanner.Migrate(reg, "/workspace")
			if err != nil {
				return err
			}

			for _, r := range results {
				if r.Success {
					fmt.Printf("migrated: %s\n", r.Path)
				} else {
					fmt.Fprintf(os.Stderr, "failed: %s: %v\n", r.Path, r.Error)
				}
			}
			return nil
		},
	}
}

func runCmd() *cobra.Command {
	var dryRun bool
	var onConflict string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Assemble, deploy, and exec OMP",
		Long:  "Resolves the consumer from cwd, composes configuration, deploys to OMP paths, then execs omp.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			gitRoot := findGitRoot(cwd)
			if gitRoot == "" {
				return fmt.Errorf("not in a git repository; run 'carrel discover' to see available repos")
			}

			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := reg.ResolveConsumer(gitRoot)
			if err != nil {
				return fmt.Errorf("unregistered repo %s; run 'carrel discover' to see available repos", gitRoot)
			}

			sources, err := reg.ResolveSources(c.ID)
			if err != nil {
				return fmt.Errorf("resolve sources: %w", err)
			}

			var sourceIDs []uuid.UUID
			for _, s := range sources {
				sourceIDs = append(sourceIDs, s.ID)
			}
			entries, err := reg.ResolveEntries(sourceIDs)
			if err != nil {
				return fmt.Errorf("resolve entries: %w", err)
			}

			plan, err := composer.Compose(c, sources, entries)
			if err != nil {
				return fmt.Errorf("compose: %w", err)
			}

			// Resolve content for each output file
			for i := range plan.Files {
				if err := resolveOutputContent(&plan.Files[i], entries, sources); err != nil {
					return fmt.Errorf("resolve content for %s: %w", plan.Files[i].DestinationPath, err)
				}
			}

			// Determine conflict policy
			var policy deployer.ConflictPolicy
			switch onConflict {
			case "backup":
				policy = deployer.ConflictBackup
			case "skip":
				policy = deployer.ConflictSkip
			default:
				policy = deployer.ConflictError
			}


			// Make destination paths absolute for this consumer
			ompDir := filepath.Join(c.Path, ".omp")
			for i := range plan.Files {
				plan.Files[i].DestinationPath = filepath.Join(ompDir, plan.Files[i].DestinationPath)
			}

			if dryRun {
				collisions, _ := deployer.Deploy(plan, nil, policy, true)
				fmt.Printf("dry-run: %d files would be deployed\n", len(plan.Files))
				for _, col := range collisions {
					fmt.Printf("  collision: %s (%v)\n", col.Path, col.Action)
				}
				return nil
			}

			_, prevEntries, _ := reg.LastDeployment(c.ID)

			collisions, err := deployer.Deploy(plan, prevEntries, policy, false)
			now := time.Now()
			if err != nil {
				reg.RecordDeployment(registry.Deployment{
					ID: uuid.New(), ConsumerID: c.ID, AttemptedAt: now,
				}, nil)
				return err
			}

			// Record successful deployment
			var depEntries []registry.DeploymentEntry
			for _, f := range plan.Files {
				for _, eID := range f.SourceEntries {
					depEntries = append(depEntries, registry.DeploymentEntry{
						DeploymentID: uuid.New(),
						Path:         f.DestinationPath,
						ContentHash:  f.ContentHash,
						SourceEntry:  eID,
					})
				}
			}
			reg.RecordDeployment(registry.Deployment{
				ID: uuid.New(), ConsumerID: c.ID, AttemptedAt: now,
				SucceededAt: &now,
			}, depEntries)

			_ = collisions
			fmt.Printf("deployment complete — %d files written\n", len(plan.Files))

			// Exec omp
			ompBin, err := exec.LookPath("omp")
			if err != nil {
				return fmt.Errorf("omp not found in PATH")
			}
			return syscall.Exec(ompBin, []string{"omp"}, os.Environ())
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deployed without writing")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "error", "Conflict policy: error, backup, skip")
	return cmd
}

func osSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "os-setup",
		Short: "Deploy OS tool configuration in container",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("os-setup: deploying zsh, nvim, wezterm config")
			return nil
		},
	}
}

func hostSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "host-setup",
		Short: "Deploy OS tool configuration on host",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("host-setup: deploying host tool config")
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show registry state",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			consumers, err := reg.ListConsumers()
			if err != nil {
				return err
			}
			sources, err := reg.ListSources()
			if err != nil {
				return err
			}

			fmt.Println("Consumers:")
			for _, c := range consumers {
				fmt.Printf("  %s (%s) — %s\n", c.Alias, c.Path, kindStr(c.Kind))
			}
			fmt.Println("\nSources:")
			for _, s := range sources {
				fmt.Printf("  %s — %s [%s]\n", s.Alias, s.Path, scopeStr(s.Scope))
			}
			return nil
		},
	}
}

func verifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Check deployed state against claims",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			consumers, _ := reg.ListConsumers()
			for _, c := range consumers {
				_, _, err := reg.LastDeployment(c.ID)
				if err == registry.ErrNoDeployment {
					fmt.Printf("%s: no deployments\n", c.Alias)
				} else if err != nil {
					fmt.Printf("%s: error: %v\n", c.Alias, err)
				} else {
					fmt.Printf("%s: deployed\n", c.Alias)
				}
			}
			return nil
		},
	}
}

func dashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Launch TUI dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			m := tui.NewModel(reg)
			p := tea.NewProgram(m)
			if _, err := p.Run(); err != nil {
				return err
			}
			return nil
		},
	}
}

func findGitRoot(cwd string) string {
	for {
		if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
			return cwd
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return ""
		}
		cwd = parent
	}
}

func kindStr(k registry.ConsumerKind) string {
	switch k {
	case registry.ConsumerRepo:
		return "repo"
	case registry.ConsumerContainer:
		return "container"
	case registry.ConsumerHost:
		return "host"
	}
	return "unknown"
}

func scopeStr(s registry.SourceScope) string {
	switch s {
	case registry.ScopeUniversal:
		return "universal"
	case registry.ScopeTargetSpecific:
		return "target-specific"
	case registry.ScopeUserPersonal:
		return "user-personal"
	case registry.ScopeHostLocal:
		return "host-local"
	}
	return "unknown"
}
