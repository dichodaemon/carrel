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
	rootCmd.AddGroup(&cobra.Group{ID: "query", Title: "Query Commands:"})

	// Lifecycle commands
	for _, f := range []func() *cobra.Command{bootstrapCmd, discoverCmd, runCmd, osSetupCmd, hostSetupCmd} {
		cmd := f()
		cmd.GroupID = "lifecycle"
		rootCmd.AddCommand(cmd)
	}
	// Query commands
	for _, f := range []func() *cobra.Command{statusCmd, verifyCmd, dashboardCmd, sourcesCmd, deployedCmd, planCmd, previewCmd, dependentsCmd, feedsCmd, inspectCmd, traceCmd} {
		cmd := f()
		cmd.GroupID = "query"
		rootCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(localCmd())
	rootCmd.AddCommand(slotCmd())
	rootCmd.AddCommand(registerCmd())


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
					flags += " [carula config]"
				}
				fmt.Printf("%-40s %s%s\n", r.Path, status, flags)
			}
			return nil
		},
	}
}


func scanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan <source-alias>",
		Short: "Register files from a source directory as entries",
		Long:  "Walks a registered source directory, matches files to capability type conventions, and registers them as entries. Idempotent — skips already-registered entries.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			sources, err := reg.ListSources()
			if err != nil {
				return err
			}
			var source *registry.Source
			for _, s := range sources {
				if s.Alias == args[0] {
					source = &s
					break
				}
			}
			if source == nil {
				return fmt.Errorf("source %q not found", args[0])
			}

			results, err := scanner.ScanSource(reg, *source)
			if err != nil {
				return err
			}
			for _, r := range results {
				if r.Action == "error" {
					fmt.Printf("%-12s %-15s %-20s error: %v\n", r.Action, r.Type, r.Name, r.Error)
				} else {
					fmt.Printf("%-12s %-15s %s\n", r.Action, r.Type, r.Name)
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
				return fmt.Errorf("unregistered repo %s\n\nTo register, run:\n  carrel register consumer <alias> --path=%s\n  carrel register source <alias> --path=<source-dir> --scope=target --consumer=<alias>", gitRoot, gitRoot)
			}


			// Check slots.yml consistency
			for _, w := range checkSlotsConsistency(reg, c.Alias) {
				fmt.Fprintf(os.Stderr, "warning: slots.yml: %s\n", w)
			}

			// Apply source slot-defaults.yml for universal sources if consumer has no slots.yml
			consumerSlotsPath := filepath.Join(c.Path, ".carrel", "slots.yml")
			if _, err := os.Stat(consumerSlotsPath); os.IsNotExist(err) {
				sources, srcErr := reg.ResolveSources(c.ID)
				if srcErr == nil {
					for _, s := range sources {
						defaultsPath := filepath.Join(s.Path, "slot-defaults.yml")
						w, e := registry.ApplySlotDefaults(reg, c, s, defaultsPath)
						if e != nil {
							fmt.Fprintf(os.Stderr, "warning: slot-defaults.yml for %s: %v\n", s.Alias, e)
						}
						for _, ww := range w {
							fmt.Fprintf(os.Stderr, "warning: slot-defaults.yml: %s\n", ww)
						}
					}
				}
			}
			deployedPaths, err := deployConsumer(reg, c, dryRun, onConflict)
			if err != nil {
				return err
			}

			if dryRun {
				return nil
			}

			// Ensure .git/info/exclude for non-opt-in repos
			_ = deployer.EnsureGitExclude(gitRoot, c.DeployRoot, deployedPaths)

			// Release the registry lock before exec replaces the process.
			// defer reg.Close() is dead code after syscall.Exec succeeds —
			// the lock fd would be inherited by OMP, blocking all registry writes.
			reg.Close()

			// Exec omp
			ompBin, err := exec.LookPath("omp")
			if err != nil {
				return fmt.Errorf("omp not found in PATH")
			}
			return syscall.Exec(ompBin, []string{"omp"}, os.Environ())
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deployed without writing")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "error", "Conflict policy: error, backup, skip, overwrite")
	return cmd
}

func osSetupCmd() *cobra.Command {
	var dryRun bool
	var onConflict string

	cmd := &cobra.Command{
		Use:   "os-setup",
		Short: "Deploy OS tool configuration in container",
		Long:  "Resolves the container consumer, composes OS configuration (zsh, nvim, wezterm), and deploys to OS paths under /home/dev.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := reg.ResolveConsumer("container")
			if err != nil {
				return fmt.Errorf("container consumer not found; run 'carrel bootstrap'")
			}

			_, err = deployConsumer(reg, c, dryRun, onConflict)
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deployed without writing")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "error", "Conflict policy: error, backup, skip, overwrite")
	return cmd
}

func deployConsumer(reg registry.Registry, c registry.Consumer, dryRun bool, onConflict string) ([]string, error) {
	rSlots, err := reg.ResolveSlots(c.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve slots: %w", err)
	}

	var cSlots []composer.Slot
	entrySlots := make(map[uuid.UUID][]composer.Entry)
	entryLookup := make(map[uuid.UUID]registry.Entry)
	sourceByID := make(map[uuid.UUID]registry.Source)

	for _, s := range rSlots {
		dest := filepath.Clean(c.DeployRoot + "/" + s.DestPath)
		cSlots = append(cSlots, composer.Slot{
			ID:              s.ID,
			ConsumerID:      s.ConsumerID,
			Name:            s.Name,
			DestinationPath: dest,
		})

		sEntries, eErr := reg.ResolveEntrySlots(s.ID)
		if eErr != nil {
			continue
		}
		var cEntries []composer.Entry
		for _, se := range sEntries {
			cEntries = append(cEntries, composer.Entry{
				ID:       se.ID,
				SourceID: se.SourceID,
				Mode:     composer.ComposeMode(se.ComposeMode),
				Final:    se.Final,
				Priority: se.Priority,
			})
			entryLookup[se.ID] = se.Entry
		}
		entrySlots[s.ID] = cEntries
	}

	plan, err := composer.Compose(c.ID, cSlots, entrySlots)
	if err != nil {
		return nil, fmt.Errorf("compose: %w", err)
	}

	sources, _ := reg.ListSources()
	for _, s := range sources {
		sourceByID[s.ID] = s
	}

	for i := range plan.Files {
		resolveOutputContent(&plan.Files[i], entryLookup, sourceByID)
	}

	var policy deployer.ConflictPolicy
	switch onConflict {
	case "backup":
		policy = deployer.ConflictBackup
	case "skip":
		policy = deployer.ConflictSkip
	case "overwrite":
		policy = deployer.ConflictOverwrite
	default:
		policy = deployer.ConflictError
	}


	// Collect deployed paths for git exclude
	var deployedPaths []string
	for _, f := range plan.Files {
		deployedPaths = append(deployedPaths, f.DestinationPath)
	}
	if dryRun {
		collisions, _ := deployer.Deploy(plan, nil, policy, true)
		fmt.Printf("dry-run: %d files would be deployed\n", len(plan.Files))
		for _, col := range collisions {
			fmt.Printf("  collision: %s (%v)\n", col.Path, col.Action)
		}
		return deployedPaths, nil
	}

	_, prevEntries, _ := reg.LastDeployment(c.ID)
	now := time.Now()

	collisions, err := deployer.Deploy(plan, prevEntries, policy, false)
	if err != nil {
		reg.RecordDeployment(registry.Deployment{
			ID: uuid.New(), ConsumerID: c.ID, AttemptedAt: now,
		}, nil)
		return nil, err
	}

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
	return deployedPaths, nil
}

func hostSetupCmd() *cobra.Command {
	var dryRun bool
	var onConflict string

	cmd := &cobra.Command{
		Use:   "host-setup",
		Short: "Deploy OS tool configuration on host",
		Long:  "Resolves the host consumer, composes OS configuration, and deploys to host OS paths.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := mustOpenRegistry()
			defer reg.Close()

			c, err := reg.ResolveConsumer("host")
			if err != nil {
				return fmt.Errorf("host consumer not found; run 'carrel bootstrap'")
			}

			_, err = deployConsumer(reg, c, dryRun, onConflict)
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deployed without writing")
	cmd.Flags().StringVar(&onConflict, "on-conflict", "error", "Conflict policy: error, backup, skip, overwrite")
	return cmd
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
