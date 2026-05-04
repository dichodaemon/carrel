package main

import (
	"fmt"
	"os"

	"github.com/dichodaemon/carrel/internal/registry"
)

func mustOpenRegistry() *registry.DoltRegistry {
	if err := os.MkdirAll("/workspace/.carrel", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "registry dir: %v\n", err)
		os.Exit(1)
	}
	dsn := "file:///workspace/.carrel?commitname=Carrel&commitemail=carrel@localhost&database=registry"
	reg, err := registry.NewDoltRegistry(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(1)
	}
	return reg
}
