package main

import (
	"fmt"
	"os"

	"github.com/dichodaemon/carrel/internal/registry"
)

func mustOpenRegistry() registry.Registry {
	dir := os.Getenv("CARREL_REGISTRY_DIR")
	if dir == "" {
		dir = "/workspace/.carrel"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "registry dir: %v\n", err)
		os.Exit(1)
	}
	reg, err := registry.New(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(1)
	}
	return reg
}
