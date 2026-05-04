package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/dichodaemon/carrel/internal/registry"
)

func runBootstrap(cmd *cobra.Command, args []string) error {
	registryDir := "/workspace/.carrel"
	dsn := fmt.Sprintf("file://%s?commitname=Carrel&commitemail=carrel@localhost&database=registry", registryDir)

	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return fmt.Errorf("bootstrap: create registry dir: %w", err)
	}

	reg, err := registry.NewDoltRegistry(dsn)
	if err != nil {
		return fmt.Errorf("bootstrap: open registry: %w", err)
	}
	defer reg.Close()

	if err := registry.ApplyMigrations(reg); err != nil {
		return fmt.Errorf("bootstrap: apply migrations: %w", err)
	}

	carrelConsumer := registry.Consumer{
		ID:    uuid.New(),
		Alias: "carrel",
		Path:  "/workspace/carrel",
		Kind:  registry.ConsumerRepo,
	}
	_ = reg.RegisterConsumer(carrelConsumer)

	folioConsumer := registry.Consumer{
		ID:    uuid.New(),
		Alias: "folio",
		Path:  "/workspace/folio",
		Kind:  registry.ConsumerRepo,
	}
	_ = reg.RegisterConsumer(folioConsumer)

	carrelSource := registry.Source{
		ID:    uuid.New(),
		Alias: "carrel-omp",
		Path:  "/workspace/carrel/omp",
		Scope: registry.ScopeUniversal,
		Kind:  registry.SourceGitBacked,
	}
	_ = reg.RegisterSource(carrelSource)

	fmt.Println("bootstrap: registry initialized")
	return nil
}
